# Queue System Documentation

Complete guide to using the database-backed queue system for background job processing.

## Table of Contents

- [What is the Queue System](#what-is-the-queue-system)
- [When to Use It](#when-to-use-it)
- [Architecture](#architecture)
- [Task Types](#task-types)
- [Queue Manager API](#queue-manager-api)
- [Progress Tracking](#progress-tracking)
- [Retry Logic](#retry-logic)
- [Worker Pattern](#worker-pattern)
- [Examples](#examples)
- [Best Practices](#best-practices)
- [Monitoring and Maintenance](#monitoring-and-maintenance)

## What is the Queue System

The queue system is a **database-backed asynchronous task processing system** that allows you to:

- Offload long-running operations from HTTP requests
- Track progress of background jobs
- Automatically retry failed tasks
- Process tasks with multiple workers
- Provide real-time feedback to users

### Key Features

- **PostgreSQL-backed** - Durable, transactional task storage
- **Progress tracking** - Update 0-100% progress for UI feedback
- **Automatic retries** - Configurable retry attempts for failed tasks
- **Worker coordination** - `SKIP LOCKED` prevents duplicate processing
- **Status tracking** - pending → in_progress → completed/failed
- **JSONB payloads** - Flexible task data storage

## When to Use It

Use the queue system for operations that:

### 1. Take More Than 2 Seconds

Don't make users wait for slow operations:

```go
// ❌ BAD: Blocks HTTP request for 30 seconds
func (mr *mutationResolver) ExportData(ctx context.Context, projectID string) (*Export, error) {
    data := fetchAllData(projectID)        // 10 seconds
    file := generateCSV(data)              // 15 seconds
    url := uploadToStorage(file)           // 5 seconds
    return &Export{URL: url}, nil
}

// ✅ GOOD: Returns immediately, processes in background
func (mr *mutationResolver) ExportData(ctx context.Context, projectID string) (*gen.Task, error) {
    payload := map[string]string{"project_id": projectID}
    taskID, err := mr.QueueManager.Enqueue(ctx, queue.TaskTypeDataExport, payload)

    return &gen.Task{
        ID:       fmt.Sprintf("%d", taskID),
        Status:   "PENDING",
        Progress: 0,
    }, nil
}
```

### 2. Batch Operations

Processing multiple items:

```go
// Import 10,000 records
// Process 5,000 images
// Send 1,000 emails
// Generate reports for 500 projects
```

### 3. Operations That Can Fail

When retry logic is beneficial:

```go
// External API calls (may timeout)
// File uploads (network issues)
// Email sending (rate limits)
// Webhook deliveries (recipient down)
```

### 4. Non-Critical Operations

When immediate completion isn't required:

```go
// Generate analytics reports
// Send notification emails
// Clean up old data
// Sync with external systems
```

### 5. Operations Requiring Progress Feedback

When users need to see progress:

```go
// "Importing... 45%"
// "Generating report... 80%"
// "Processing images... 23 of 100"
```

## Architecture

### Task Lifecycle

```
┌─────────────────────────────────────────────────────────┐
│ 1. ENQUEUE (Resolver)                                   │
│    Client request → Resolver → QueueManager.Enqueue()  │
│    • Create task record in database                    │
│    • Status: PENDING, Progress: 0                      │
│    • Store payload as JSONB                            │
│    • Return task ID to client                          │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ 2. CLIENT RETURNS IMMEDIATELY                           │
│    Response: { taskID: "123", status: "PENDING" }      │
│    Client can poll for progress                         │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ 3. DEQUEUE (Worker)                                     │
│    Worker polls → QueueManager.Dequeue()               │
│    • SELECT ... FOR UPDATE SKIP LOCKED                 │
│    • Update status: IN_PROGRESS                        │
│    • Set started_at timestamp                          │
│    • Return task to worker                             │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ 4. PROCESS (Service)                                    │
│    Worker → Service.ProcessTask()                      │
│    • Load task payload                                 │
│    • Process items                                     │
│    • Update progress periodically (25%, 50%, 75%)      │
│    • Handle errors gracefully                          │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ 5. COMPLETE or FAIL                                     │
│                                                         │
│    SUCCESS:                                             │
│    • QueueManager.CompleteTask()                       │
│    • Status: COMPLETED, Progress: 100                  │
│    • Set completed_at timestamp                        │
│                                                         │
│    FAILURE:                                             │
│    • QueueManager.FailTask()                           │
│    • Increment retry_count                             │
│    • If retry_count < max_retries:                     │
│    │   Status: PENDING (will retry)                    │
│    • Else:                                             │
│    │   Status: FAILED (permanent)                      │
│    • Store error_message                               │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ 6. CLIENT POLLING                                       │
│    Client polls: task(id: "123") { status, progress }  │
│    • PENDING → 0%                                       │
│    • IN_PROGRESS → 50%                                  │
│    • COMPLETED → 100%                                   │
└─────────────────────────────────────────────────────────┘
```

### Database Schema

```sql
CREATE TABLE tasks (
    id SERIAL PRIMARY KEY,
    payload JSONB NOT NULL,
    type TEXT NOT NULL,
    progress INTEGER DEFAULT 0,
    status TEXT DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3
);

-- Index for efficient dequeue
CREATE INDEX idx_tasks_status_type ON tasks(status, type, created_at);
```

## Task Types

Define task types as constants:

```go
// res/queue/queue.go
type TaskType string

const (
    TaskTypeEmailNotification TaskType = "email_notification"
    TaskTypeDataExport        TaskType = "data_export"
    TaskTypeBackgroundJob     TaskType = "background_job"
)
```

Add your own task types:

```go
const (
    TaskTypeImageProcessing   TaskType = "image_processing"
    TaskTypeReportGeneration  TaskType = "report_generation"
    TaskTypeBulkImport        TaskType = "bulk_import"
    TaskTypeWebhookDelivery   TaskType = "webhook_delivery"
)
```

## Queue Manager API

### Initialize Queue Manager

```go
// In main.go or api.go
db := setupDatabase()
queueManager := queue.NewManager(db)
```

### Enqueue

Add a new task to the queue:

```go
func (m *Manager) Enqueue(
    ctx context.Context,
    taskType TaskType,
    payload interface{},
) (int, error)
```

**Example:**

```go
payload := map[string]string{
    "project_id": projectID,
    "user_id":    currentUser.ID,
    "format":     "csv",
}

taskID, err := mr.QueueManager.Enqueue(
    ctx,
    queue.TaskTypeDataExport,
    payload,
)
if err != nil {
    return nil, errors.New("failed to start export")
}

return &gen.Task{ID: fmt.Sprintf("%d", taskID), Status: "PENDING"}, nil
```

### EnqueueIfNotExists

Prevent duplicate tasks for the same resource:

```go
func (m *Manager) EnqueueIfNotExists(
    ctx context.Context,
    taskType TaskType,
    payload interface{},
) (int, bool, error)
```

**Example:**

```go
payload := map[string]string{
    "project_id": projectID,
}

taskID, created, err := mr.QueueManager.EnqueueIfNotExists(
    ctx,
    queue.TaskTypeImageProcessing,
    payload,
)

if !created {
    // Task already exists and is pending/in_progress
    return &gen.Task{
        ID:     fmt.Sprintf("%d", taskID),
        Status: "ALREADY_PROCESSING",
    }, nil
}

return &gen.Task{ID: fmt.Sprintf("%d", taskID), Status: "PENDING"}, nil
```

### Dequeue

Worker retrieves next pending task:

```go
func (m *Manager) Dequeue(
    ctx context.Context,
    taskType TaskType,
) (*Task, error)
```

**Example:**

```go
// In worker
for {
    task, err := queueManager.Dequeue(ctx, queue.TaskTypeDataExport)
    if err != nil {
        log.Printf("Dequeue error: %v", err)
        time.Sleep(5 * time.Second)
        continue
    }

    if task == nil {
        // No tasks available
        time.Sleep(5 * time.Second)
        continue
    }

    // Process task
    processTask(task)
}
```

### UpdateProgress

Update task progress (0-100):

```go
func (m *Manager) UpdateProgress(
    ctx context.Context,
    taskID int,
    progress int,
) error
```

**Example:**

```go
total := len(items)
for i, item := range items {
    processItem(item)

    // Update progress every 10 items or at milestones
    if i%10 == 0 || i == total-1 {
        progress := int(float64(i+1) / float64(total) * 100)
        queueManager.UpdateProgress(ctx, task.ID, progress)
    }
}
```

### CompleteTask

Mark task as successfully completed:

```go
func (m *Manager) CompleteTask(
    ctx context.Context,
    taskID int,
) error
```

**Example:**

```go
// After successful processing
err := queueManager.CompleteTask(ctx, task.ID)
if err != nil {
    log.Printf("Failed to mark task complete: %v", err)
}
```

### FailTask

Mark task as failed (with retry logic):

```go
func (m *Manager) FailTask(
    ctx context.Context,
    taskID int,
    errorMsg string,
) error
```

**Example:**

```go
err := processTask(task)
if err != nil {
    errMsg := fmt.Sprintf("Processing failed: %v", err)
    queueManager.FailTask(ctx, task.ID, errMsg)
    log.Printf("Task failed: %s", errMsg)
    return
}
```

### GetTask

Retrieve task by ID:

```go
func (m *Manager) GetTask(
    ctx context.Context,
    taskID int,
) (*Task, error)
```

**Example:**

```go
// In GraphQL resolver
func (qr *queryResolver) Task(ctx context.Context, id string) (*gen.Task, error) {
    taskID, err := strconv.Atoi(id)
    if err != nil {
        return nil, errors.New("invalid task ID")
    }

    task, err := qr.QueueManager.GetTask(ctx, taskID)
    if err != nil {
        return nil, errors.New("task not found")
    }

    return &gen.Task{
        ID:       fmt.Sprintf("%d", task.ID),
        Status:   string(task.Status),
        Progress: task.Progress,
    }, nil
}
```

### GetTaskProgress

Get progress for a specific task type and project:

```go
func (m *Manager) GetTaskProgress(
    ctx context.Context,
    taskType TaskType,
    projectID string,
) (*int, error)
```

**Example:**

```go
progress, err := queueManager.GetTaskProgress(
    ctx,
    queue.TaskTypeDataExport,
    projectID,
)

if progress != nil {
    return &gen.TaskProgress{
        ProjectID: projectID,
        Progress:  *progress,
        Active:    true,
    }, nil
}

return nil, nil // No active task
```

### CleanupOldTasks

Remove old completed tasks:

```go
func (m *Manager) CleanupOldTasks(
    ctx context.Context,
    olderThan time.Duration,
) (int64, error)
```

**Example:**

```go
// Run daily cleanup job
deleted, err := queueManager.CleanupOldTasks(ctx, 7*24*time.Hour) // 7 days
log.Printf("Cleaned up %d old tasks", deleted)
```

## Progress Tracking

### Why Track Progress

Users love feedback on long-running operations:

```
❌ "Processing..." (no feedback, user anxious)
✅ "Processing... 45%" (clear feedback, user calm)
```

### How to Track Progress

```go
func (s *service) ProcessDataExport(ctx context.Context, task *queue.Task) error {
    // 1. Parse payload
    var payload struct {
        ProjectID string `json:"project_id"`
    }
    json.Unmarshal(task.Payload, &payload)

    // 2. Fetch items
    items, err := s.store.Items().ListByProject(ctx, payload.ProjectID)
    if err != nil {
        return fmt.Errorf("failed to fetch items: %w", err)
    }

    total := len(items)
    processed := 0

    // 3. Process with progress updates
    for _, item := range items {
        // Process item
        if err := s.processItem(ctx, item); err != nil {
            s.logger.Printf("Failed to process item %s: %v", item.ID, err)
            // Continue processing other items
            continue
        }

        processed++

        // Update progress every 10 items or at key milestones
        if processed%10 == 0 || processed == total {
            progress := int(float64(processed) / float64(total) * 100)

            err := s.queueManager.UpdateProgress(ctx, task.ID, progress)
            if err != nil {
                s.logger.Printf("Failed to update progress: %v", err)
                // Non-fatal, continue processing
            }
        }
    }

    // 4. Final step - mark complete
    return s.queueManager.CompleteTask(ctx, task.ID)
}
```

### Progress Best Practices

1. **Update at meaningful intervals** - Don't update every item if processing thousands
2. **Use meaningful percentages** - Break work into phases (fetch 25%, process 50%, upload 75%, complete 100%)
3. **Don't block on updates** - Progress updates should be non-blocking
4. **Log failures** - If progress update fails, log but continue processing

### Multi-Phase Progress

For complex operations with multiple phases:

```go
func (s *service) ProcessComplexTask(ctx context.Context, task *queue.Task) error {
    // Phase 1: Fetch data (0-25%)
    s.queueManager.UpdateProgress(ctx, task.ID, 5)
    data, err := s.fetchData(ctx)
    if err != nil {
        return err
    }
    s.queueManager.UpdateProgress(ctx, task.ID, 25)

    // Phase 2: Transform data (25-50%)
    s.queueManager.UpdateProgress(ctx, task.ID, 30)
    transformed := s.transformData(data)
    s.queueManager.UpdateProgress(ctx, task.ID, 50)

    // Phase 3: Upload to storage (50-75%)
    s.queueManager.UpdateProgress(ctx, task.ID, 55)
    err = s.uploadData(ctx, transformed)
    if err != nil {
        return err
    }
    s.queueManager.UpdateProgress(ctx, task.ID, 75)

    // Phase 4: Send notifications (75-100%)
    s.queueManager.UpdateProgress(ctx, task.ID, 80)
    s.sendNotifications(ctx)
    s.queueManager.UpdateProgress(ctx, task.ID, 90)

    // Complete
    return s.queueManager.CompleteTask(ctx, task.ID)
}
```

## Retry Logic

### Automatic Retries

Failed tasks are automatically retried up to `max_retries` times:

```go
type Task struct {
    RetryCount  int  `gorm:"type:integer;default:0"`
    MaxRetries  int  `gorm:"type:integer;default:3"`
}
```

### How Retry Works

```go
func (m *Manager) FailTask(ctx context.Context, taskID int, errorMsg string) error {
    // Atomic retry logic
    result := m.db.WithContext(ctx).Exec(`
        UPDATE tasks
        SET status = CASE
                WHEN retry_count < max_retries THEN ?  -- 'pending'
                ELSE ?                                  -- 'failed'
            END,
            retry_count = retry_count + 1,
            error_message = ?,
            started_at = NULL
        WHERE id = ?
    `, TaskStatusPending, TaskStatusFailed, errorMsg, taskID)

    return result.Error
}
```

**Flow:**
1. Task fails → `FailTask()` called
2. If `retry_count < max_retries`:
   - Status: `PENDING` (will be retried)
   - `retry_count++`
   - `started_at = NULL`
3. Else:
   - Status: `FAILED` (permanently failed)
   - Store error message

### Configuring Retries

Set custom `max_retries` when enqueuing:

```go
// Create task with custom retries
task := &queue.Task{
    Type:       queue.TaskTypeWebhookDelivery,
    Payload:    payloadJSON,
    Status:     queue.TaskStatusPending,
    MaxRetries: 5,  // Custom retry count
}
db.Create(task)
```

### Exponential Backoff

Implement backoff in worker:

```go
func processWithBackoff(task *queue.Task) error {
    delay := time.Duration(math.Pow(2, float64(task.RetryCount))) * time.Second

    if task.RetryCount > 0 {
        log.Printf("Retry #%d, waiting %v", task.RetryCount, delay)
        time.Sleep(delay)
    }

    return processTask(task)
}
```

## Worker Pattern

Workers are NOT included in the template but here's how to implement them:

### Simple Worker

```go
// cmd/worker/main.go
package main

import (
    "context"
    "log"
    "os"
    "time"

    "saas-starter-api/res/queue"
    "saas-starter-api/res/store/postgresql"
)

func main() {
    db := setupDatabase()
    queueManager := queue.NewManager(db)
    store := postgresql.NewStore(db)

    logger := log.New(os.Stdout, "[WORKER] ", log.LstdFlags)

    // Process tasks
    for {
        ctx := context.Background()

        task, err := queueManager.Dequeue(ctx, queue.TaskTypeDataExport)
        if err != nil {
            logger.Printf("Dequeue error: %v", err)
            time.Sleep(5 * time.Second)
            continue
        }

        if task == nil {
            // No tasks available
            time.Sleep(5 * time.Second)
            continue
        }

        logger.Printf("Processing task %d", task.ID)

        // Process task
        err = processTask(ctx, task, store, queueManager, logger)
        if err != nil {
            logger.Printf("Task %d failed: %v", task.ID, err)
            queueManager.FailTask(ctx, task.ID, err.Error())
        } else {
            logger.Printf("Task %d completed", task.ID)
        }
    }
}

func processTask(
    ctx context.Context,
    task *queue.Task,
    store store.Store,
    queueManager *queue.Manager,
    logger *log.Logger,
) error {
    switch task.Type {
    case queue.TaskTypeDataExport:
        return processDataExport(ctx, task, store, queueManager, logger)
    case queue.TaskTypeImageProcessing:
        return processImages(ctx, task, store, queueManager, logger)
    default:
        return fmt.Errorf("unknown task type: %s", task.Type)
    }
}
```

### Multi-Worker with Concurrency

```go
func main() {
    db := setupDatabase()
    queueManager := queue.NewManager(db)

    // Run 5 workers concurrently
    numWorkers := 5
    for i := 0; i < numWorkers; i++ {
        go startWorker(i, queueManager)
    }

    // Keep main alive
    select {}
}

func startWorker(id int, queueManager *queue.Manager) {
    logger := log.New(os.Stdout, fmt.Sprintf("[WORKER-%d] ", id), log.LstdFlags)

    for {
        ctx := context.Background()

        // SKIP LOCKED prevents multiple workers from processing same task
        task, err := queueManager.Dequeue(ctx, queue.TaskTypeDataExport)
        if err != nil {
            logger.Printf("Error: %v", err)
            time.Sleep(5 * time.Second)
            continue
        }

        if task == nil {
            time.Sleep(5 * time.Second)
            continue
        }

        logger.Printf("Processing task %d", task.ID)
        processTask(ctx, task, queueManager, logger)
    }
}
```

### Graceful Shutdown

```go
func main() {
    db := setupDatabase()
    queueManager := queue.NewManager(db)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle shutdown signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    go func() {
        <-sigChan
        log.Println("Shutting down gracefully...")
        cancel()
    }()

    // Worker with context
    for {
        select {
        case <-ctx.Done():
            log.Println("Worker stopped")
            return
        default:
            task, err := queueManager.Dequeue(ctx, queue.TaskTypeDataExport)
            if err != nil {
                time.Sleep(5 * time.Second)
                continue
            }

            if task == nil {
                time.Sleep(5 * time.Second)
                continue
            }

            processTask(ctx, task, queueManager)
        }
    }
}
```

## Examples

### Example 1: Data Export

**Resolver:**

```go
func (mr *mutationResolver) ExportProjectData(
    ctx context.Context,
    projectID string,
) (*gen.Task, error) {
    currentUser := middleware.GetCurrentUser(ctx)
    if currentUser == nil {
        return nil, errors.New("unauthorized")
    }

    if err := mr.HasProjectAccess(ctx, projectID); err != nil {
        return nil, err
    }

    payload := map[string]string{
        "project_id": projectID,
        "user_id":    currentUser.ID,
    }

    taskID, err := mr.QueueManager.Enqueue(
        ctx,
        queue.TaskTypeDataExport,
        payload,
    )
    if err != nil {
        mr.Logger.Printf("Failed to enqueue export: %v", err)
        return nil, errors.New("failed to start export")
    }

    return &gen.Task{
        ID:       fmt.Sprintf("%d", taskID),
        Status:   "PENDING",
        Progress: 0,
    }, nil
}
```

**Service:**

```go
func (s *exportService) ProcessExport(
    ctx context.Context,
    task *queue.Task,
) error {
    var payload struct {
        ProjectID string `json:"project_id"`
        UserID    string `json:"user_id"`
    }

    if err := json.Unmarshal(task.Payload, &payload); err != nil {
        return fmt.Errorf("invalid payload: %w", err)
    }

    // Phase 1: Fetch data (0-30%)
    s.queueManager.UpdateProgress(ctx, task.ID, 10)
    items, err := s.store.Items().ListByProject(ctx, payload.ProjectID)
    if err != nil {
        return fmt.Errorf("failed to fetch items: %w", err)
    }
    s.queueManager.UpdateProgress(ctx, task.ID, 30)

    // Phase 2: Generate CSV (30-70%)
    s.queueManager.UpdateProgress(ctx, task.ID, 40)
    csvData := s.generateCSV(items)
    s.queueManager.UpdateProgress(ctx, task.ID, 70)

    // Phase 3: Upload (70-90%)
    s.queueManager.UpdateProgress(ctx, task.ID, 75)
    url, err := s.uploadToStorage(ctx, csvData)
    if err != nil {
        return fmt.Errorf("upload failed: %w", err)
    }
    s.queueManager.UpdateProgress(ctx, task.ID, 90)

    // Phase 4: Send email (90-100%)
    user, _ := s.store.Users().Get(ctx, payload.UserID)
    s.mailService.SendExportReady(ctx, user.Email, url)
    s.queueManager.UpdateProgress(ctx, task.ID, 95)

    return s.queueManager.CompleteTask(ctx, task.ID)
}
```

### Example 2: Bulk Import

**Enqueue:**

```go
func (mr *mutationResolver) ImportItems(
    ctx context.Context,
    projectID string,
    fileURL string,
) (*gen.Task, error) {
    payload := map[string]string{
        "project_id": projectID,
        "file_url":   fileURL,
    }

    // Prevent duplicate imports
    taskID, created, err := mr.QueueManager.EnqueueIfNotExists(
        ctx,
        queue.TaskTypeBulkImport,
        payload,
    )

    if !created {
        return nil, errors.New("import already in progress")
    }

    return &gen.Task{ID: fmt.Sprintf("%d", taskID), Status: "PENDING"}, nil
}
```

**Process:**

```go
func (s *importService) ProcessImport(ctx context.Context, task *queue.Task) error {
    var payload struct {
        ProjectID string `json:"project_id"`
        FileURL   string `json:"file_url"`
    }
    json.Unmarshal(task.Payload, &payload)

    // Download file
    s.queueManager.UpdateProgress(ctx, task.ID, 10)
    data, err := s.downloadFile(payload.FileURL)
    if err != nil {
        return err
    }

    // Parse CSV
    s.queueManager.UpdateProgress(ctx, task.ID, 20)
    items, err := s.parseCSV(data)
    if err != nil {
        return err
    }

    // Import items with progress
    total := len(items)
    for i, item := range items {
        if err := s.importItem(ctx, item, payload.ProjectID); err != nil {
            s.logger.Printf("Failed to import item %d: %v", i, err)
            continue // Don't fail entire import
        }

        if i%100 == 0 {
            progress := 20 + int(float64(i)/float64(total)*70)
            s.queueManager.UpdateProgress(ctx, task.ID, progress)
        }
    }

    s.queueManager.UpdateProgress(ctx, task.ID, 95)
    return s.queueManager.CompleteTask(ctx, task.ID)
}
```

## Best Practices

### 1. Always Update Progress

Users love feedback:

```go
// ✅ GOOD
for i, item := range items {
    processItem(item)
    if i%10 == 0 {
        progress := int(float64(i+1) / float64(total) * 100)
        queueManager.UpdateProgress(ctx, task.ID, progress)
    }
}

// ❌ BAD - No progress updates
for _, item := range items {
    processItem(item)
}
```

### 2. Handle Partial Failures Gracefully

Don't fail entire batch on one error:

```go
// ✅ GOOD
successCount := 0
for _, item := range items {
    if err := processItem(item); err != nil {
        logger.Printf("Item failed: %v", err)
        continue // Process other items
    }
    successCount++
}

if successCount == 0 {
    return errors.New("all items failed")
}

// ❌ BAD - First error fails entire batch
for _, item := range items {
    if err := processItem(item); err != nil {
        return err // Stops processing
    }
}
```

### 3. Log Before Failing

Always log errors before calling `FailTask`:

```go
// ✅ GOOD
err := processTask(task)
if err != nil {
    logger.Printf("Task %d failed: %v", task.ID, err)
    queueManager.FailTask(ctx, task.ID, err.Error())
    return
}

// ❌ BAD - No context on what failed
if err != nil {
    queueManager.FailTask(ctx, task.ID, "failed")
}
```

### 4. Use EnqueueIfNotExists for Idempotent Operations

Prevent duplicate tasks:

```go
// ✅ GOOD - Prevent duplicate exports
taskID, created, err := queueManager.EnqueueIfNotExists(ctx, taskType, payload)
if !created {
    return errors.New("export already in progress")
}

// ❌ BAD - Multiple exports can start
taskID, err := queueManager.Enqueue(ctx, taskType, payload)
```

### 5. Set Appropriate Retry Limits

Consider the operation type:

```go
// Quick operations - fewer retries
MaxRetries: 3  // Default for most operations

// External API calls - more retries
MaxRetries: 5  // APIs can be flaky

// Critical operations - many retries
MaxRetries: 10 // Important not to lose

// One-time operations - no retries
MaxRetries: 0  // Don't retry on failure
```

### 6. Clean Up Old Tasks

Prevent table bloat:

```go
// Run daily
func cleanupOldTasks(queueManager *queue.Manager) {
    ctx := context.Background()

    // Keep completed tasks for 7 days
    deleted, err := queueManager.CleanupOldTasks(ctx, 7*24*time.Hour)
    if err != nil {
        log.Printf("Cleanup failed: %v", err)
        return
    }

    log.Printf("Cleaned up %d old tasks", deleted)
}
```

### 7. Monitor Queue Depth

Track pending tasks:

```go
func getQueueDepth(db *gorm.DB, taskType queue.TaskType) int {
    var count int64
    db.Model(&queue.Task{}).
        Where("type = ?", taskType).
        Where("status = ?", queue.TaskStatusPending).
        Count(&count)
    return int(count)
}
```

## Monitoring and Maintenance

### Useful Queries

**Count tasks by status:**

```sql
SELECT status, COUNT(*) as count
FROM tasks
GROUP BY status;
```

**Find stuck tasks:**

```sql
SELECT id, type, started_at, retry_count
FROM tasks
WHERE status = 'in_progress'
  AND started_at < NOW() - INTERVAL '1 hour';
```

**Average processing time:**

```sql
SELECT type,
       AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) as avg_seconds
FROM tasks
WHERE status = 'completed'
  AND completed_at > NOW() - INTERVAL '24 hours'
GROUP BY type;
```

**Failed tasks with errors:**

```sql
SELECT id, type, error_message, retry_count, created_at
FROM tasks
WHERE status = 'failed'
ORDER BY created_at DESC
LIMIT 100;
```

### Alerting

Set up alerts for:

- Queue depth exceeds threshold
- Failed tasks rate increases
- Tasks stuck in `in_progress` for too long
- Worker crashes (no dequeue activity)

### Metrics to Track

- Queue depth by task type
- Average processing time
- Success/failure rate
- Retry rate
- Worker throughput

## Summary

The queue system enables:

- **Async processing** - Don't block HTTP requests
- **Progress tracking** - Real-time feedback for users
- **Automatic retries** - Handle transient failures
- **Scalability** - Add more workers as needed
- **Reliability** - Database-backed durability

Key takeaways:

- Use for operations > 2 seconds
- Always update progress
- Handle failures gracefully
- Log extensively
- Clean up old tasks
- Monitor queue health

## Next Steps

- **Read [API_REFERENCE.md](API_REFERENCE.md)** - GraphQL API documentation
- **Read [DEVELOPMENT.md](DEVELOPMENT.md)** - Development workflow
- **Study the code** - See `res/queue/queue.go`
- **Implement a worker** - Follow the worker pattern above

Queue wisely. Your users will thank you.
