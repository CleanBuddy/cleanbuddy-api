package graphql

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"cleanbuddy-api/res/store"
	"cleanbuddy-api/sys/graphql/gen"
	"cleanbuddy-api/sys/http/middleware"

	"github.com/google/uuid"
)

// FIELD RESOLVERS
type bookingResolver struct{ *Resolver }
func (r *Resolver) Booking() gen.BookingResolver { return &bookingResolver{r} }

func (br *bookingResolver) ServiceAddOns(ctx context.Context, booking *store.Booking) ([]store.ServiceAddOn, error) {
	// Parse JSON array from booking.ServiceAddOns string
	if booking.ServiceAddOns == "" {
		return []store.ServiceAddOn{}, nil
	}

	var addOns []store.ServiceAddOn
	if err := json.Unmarshal([]byte(booking.ServiceAddOns), &addOns); err != nil {
		br.Logger.Printf("Error parsing service add-ons: %s", err)
		return []store.ServiceAddOn{}, nil
	}

	return addOns, nil
}

func (br *bookingResolver) Review(ctx context.Context, booking *store.Booking) (*store.Review, error) {
	review, _ := br.Store.Reviews().GetByBooking(ctx, booking.ID)
	return review, nil
}

func (br *bookingResolver) Transaction(ctx context.Context, booking *store.Booking) (*store.Transaction, error) {
	transactions, err := br.Store.Transactions().GetByBooking(ctx, booking.ID)
	if err != nil || len(transactions) == 0 {
		return nil, nil
	}
	return transactions[0], nil
}

// QUERY RESOLVERS
func (qr *queryResolver) Booking(ctx context.Context, id string) (*store.Booking, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("authentication required")
	}

	booking, err := qr.Store.Bookings().Get(ctx, id)
	if err != nil {
		qr.Logger.Printf("Error retrieving booking: %s", err)
		return nil, errors.New("booking not found")
	}

	// Verify user has access to this booking (customer, cleaner, or admin)
	if booking.CustomerID != currentUser.ID && booking.CleanerID != currentUser.ID && !currentUser.IsGlobalAdmin() {
		return nil, errors.New("access denied")
	}

	return booking, nil
}

func (qr *queryResolver) MyBookings(ctx context.Context, filters *gen.BookingFiltersInput, limit, offset *int, orderBy *string) (*gen.BookingConnection, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("authentication required")
	}

	// Build filters
	bookingFilters := store.BookingFilters{}
	if filters != nil {
		if filters.Status != nil {
			bookingFilters.Status = (*store.BookingStatus)(filters.Status)
		}
		if filters.ServiceType != nil {
			bookingFilters.ServiceType = (*store.ServiceType)(filters.ServiceType)
		}
		if filters.StartDate != nil {
			bookingFilters.StartDate = filters.StartDate
		}
		if filters.EndDate != nil {
			bookingFilters.EndDate = filters.EndDate
		}
		if filters.IsRecurring != nil {
			bookingFilters.IsRecurring = filters.IsRecurring
		}
	}

	// Set default limit
	defaultLimit := 50
	if limit == nil {
		limit = &defaultLimit
	}
	defaultOffset := 0
	if offset == nil {
		offset = &defaultOffset
	}

	// Get bookings for customer
	bookings, err := qr.Store.Bookings().GetByCustomer(ctx, currentUser.ID, bookingFilters)
	if err != nil {
		qr.Logger.Printf("Error retrieving bookings: %s", err)
		return nil, errors.New("error retrieving bookings")
	}

	// Apply pagination
	totalCount := len(bookings)
	start := *offset
	if start > totalCount {
		start = totalCount
	}
	end := start + *limit
	if end > totalCount {
		end = totalCount
	}
	paginatedBookings := bookings[start:end]

	// Build response
	edges := make([]*gen.BookingEdge, len(paginatedBookings))
	for i, booking := range paginatedBookings {
		edges[i] = &gen.BookingEdge{
			Node:   booking,
			Cursor: booking.ID,
		}
	}

	return &gen.BookingConnection{
		Edges:      edges,
		TotalCount: totalCount,
	}, nil
}

func (qr *queryResolver) MyJobs(ctx context.Context, filters *gen.BookingFiltersInput, limit, offset *int, orderBy *string) (*gen.BookingConnection, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("authentication required")
	}

	// Verify user is a cleaner
	if !currentUser.IsCleaner() && !currentUser.IsGlobalAdmin() {
		return nil, errors.New("only cleaners can view jobs")
	}

	// Build filters
	bookingFilters := store.BookingFilters{}
	if filters != nil {
		if filters.Status != nil {
			bookingFilters.Status = (*store.BookingStatus)(filters.Status)
		}
		if filters.ServiceType != nil {
			bookingFilters.ServiceType = (*store.ServiceType)(filters.ServiceType)
		}
		if filters.StartDate != nil {
			bookingFilters.StartDate = filters.StartDate
		}
		if filters.EndDate != nil {
			bookingFilters.EndDate = filters.EndDate
		}
		if filters.IsRecurring != nil {
			bookingFilters.IsRecurring = filters.IsRecurring
		}
	}

	// Set default limit
	defaultLimit := 50
	if limit == nil {
		limit = &defaultLimit
	}
	defaultOffset := 0
	if offset == nil {
		offset = &defaultOffset
	}

	// Get jobs for cleaner
	bookings, err := qr.Store.Bookings().GetByCleaner(ctx, currentUser.ID, bookingFilters)
	if err != nil {
		qr.Logger.Printf("Error retrieving jobs: %s", err)
		return nil, errors.New("error retrieving jobs")
	}

	// Apply pagination
	totalCount := len(bookings)
	start := *offset
	if start > totalCount {
		start = totalCount
	}
	end := start + *limit
	if end > totalCount {
		end = totalCount
	}
	paginatedBookings := bookings[start:end]

	// Build response
	edges := make([]*gen.BookingEdge, len(paginatedBookings))
	for i, booking := range paginatedBookings {
		edges[i] = &gen.BookingEdge{
			Node:   booking,
			Cursor: booking.ID,
		}
	}

	return &gen.BookingConnection{
		Edges:      edges,
		TotalCount: totalCount,
	}, nil
}

func (qr *queryResolver) UpcomingBookings(ctx context.Context, limit *int) ([]*store.Booking, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("authentication required")
	}

	// Set default limit
	defaultLimit := 10
	if limit == nil {
		limit = &defaultLimit
	}

	// Get bookings based on user role
	var bookings []*store.Booking
	var err error

	if currentUser.IsCleaner() {
		// Get upcoming jobs for cleaner
		bookings, err = qr.Store.Bookings().GetByCleaner(ctx, currentUser.ID, store.BookingFilters{})
	} else {
		// Get upcoming bookings for customer
		bookings, err = qr.Store.Bookings().GetByCustomer(ctx, currentUser.ID, store.BookingFilters{})
	}

	if err != nil {
		qr.Logger.Printf("Error retrieving upcoming bookings: %s", err)
		return nil, errors.New("error retrieving upcoming bookings")
	}

	// Filter for upcoming bookings (not completed, cancelled, or no-show)
	// and sort by scheduled date
	upcomingBookings := []*store.Booking{}
	for _, booking := range bookings {
		if booking.Status == store.BookingStatusPending ||
		   booking.Status == store.BookingStatusConfirmed ||
		   booking.Status == store.BookingStatusInProgress {
			upcomingBookings = append(upcomingBookings, booking)
		}
	}

	// Limit results
	if len(upcomingBookings) > *limit {
		upcomingBookings = upcomingBookings[:*limit]
	}

	return upcomingBookings, nil
}

func (qr *queryResolver) AllBookings(ctx context.Context, filters *gen.BookingFiltersInput, limit, offset *int, orderBy *string) (*gen.BookingConnection, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("authentication required")
	}

	// Only admins can view all bookings
	if !currentUser.IsGlobalAdmin() {
		return nil, errors.New("admin access required")
	}

	// TODO: Implement admin all bookings query
	// This requires a store method that gets all bookings with filters
	return nil, errors.New("not yet implemented - requires store method")
}

// MUTATION RESOLVERS
func (mr *mutationResolver) CreateBooking(ctx context.Context, input gen.CreateBookingInput) (*store.Booking, error) {
	// Get current user - support both authenticated and guest bookings
	currentUser := middleware.GetCurrentUser(ctx)

	var userID string

	// Handle user creation for guest bookings
	if currentUser != nil {
		userID = currentUser.ID
	} else if input.User != nil {
		// Create new guest user account
		newUserID := uuid.New().String()
		newUser, err := mr.Store.Users().Create(
			ctx,
			newUserID,
			input.User.DisplayName,
			input.User.Email,
			store.UserRoleClient,
			nil, // No Google identity for guest users
		)
		if err != nil {
			mr.Logger.Printf("Error creating guest user: %s", err)
			return nil, errors.New("error creating user account")
		}
		userID = newUser.ID
		mr.Logger.Printf("Created guest user account: %s (%s)", newUser.Email, newUser.ID)
	} else {
		return nil, errors.New("authentication required or user details must be provided")
	}

	// Handle address - either use existing or create new
	var addressID string
	if input.AddressID != nil {
		// Verify address exists and belongs to user
		address, err := mr.Store.Addresses().Get(ctx, *input.AddressID)
		if err != nil {
			mr.Logger.Printf("Error retrieving address: %s", err)
			return nil, errors.New("address not found")
		}
		if currentUser != nil && address.UserID != userID {
			return nil, errors.New("address does not belong to user")
		}
		addressID = *input.AddressID
	} else if input.Address != nil {
		// Create new address
		lat := input.Address.Latitude
		lng := input.Address.Longitude
		address := &store.Address{
			ID:         uuid.New().String(),
			UserID:     userID,
			Street:     input.Address.Street,
			City:       input.Address.City,
			PostalCode: input.Address.PostalCode,
			Country:    input.Address.Country,
			Latitude:   &lat,
			Longitude:  &lng,
		}
		if input.Address.County != nil {
			address.County = *input.Address.County
		}
		if input.Address.GooglePlaceID != nil {
			address.GooglePlaceID = input.Address.GooglePlaceID
		}

		// Check if this is user's first address
		existingAddresses, err := mr.Store.Addresses().GetByUser(ctx, userID)
		if err != nil {
			mr.Logger.Printf("Error checking existing addresses: %s", err)
			return nil, errors.New("failed to process address")
		}
		address.IsDefault = len(existingAddresses) == 0

		if err := mr.Store.Addresses().Create(ctx, address); err != nil {
			mr.Logger.Printf("Error creating address: %s", err)
			return nil, errors.New("error creating address")
		}
		addressID = address.ID
	} else {
		return nil, errors.New("address is required")
	}

	// Fetch and validate cleaner profile
	cleanerProfile, err := mr.Store.CleanerProfiles().Get(ctx, input.CleanerProfileID)
	if err != nil {
		mr.Logger.Printf("Error retrieving cleaner profile: %s", err)
		return nil, errors.New("cleaner profile not found")
	}
	if !cleanerProfile.IsActive {
		return nil, errors.New("cleaner is not currently accepting bookings")
	}

	// Get service definition
	serviceDefinition, err := mr.Store.Services().GetServiceDefinition(ctx, input.ServiceType)
	if err != nil {
		mr.Logger.Printf("Error retrieving service definition: %s", err)
		return nil, errors.New("service definition not found")
	}
	if !serviceDefinition.IsActive {
		return nil, errors.New("service is not currently available")
	}

	// Calculate base service price
	// TODO: hourlyRate will be re-implemented later with a proper pricing system
	baseHours := serviceDefinition.BaseHours
	hourlyRate := 0 // Placeholder until pricing is re-implemented
	priceMultiplier := serviceDefinition.PriceMultiplier
	servicePrice := int(float64(hourlyRate) * baseHours * priceMultiplier)

	// Calculate add-ons price and duration
	addOnsPrice := 0
	addOnsDuration := 0.0
	if len(input.ServiceAddOns) > 0 {
		addOnDefs, err := mr.Store.Services().ListAddOnDefinitions(ctx, true)
		if err != nil {
			mr.Logger.Printf("Error retrieving add-on definitions: %s", err)
		} else {
			addOnMap := make(map[store.ServiceAddOn]*store.ServiceAddOnDefinition)
			for _, def := range addOnDefs {
				addOnMap[def.AddOn] = def
			}
			for _, addOn := range input.ServiceAddOns {
				if def, exists := addOnMap[addOn]; exists {
					addOnsPrice += def.FixedPrice
					addOnsDuration += def.EstimatedHours
				}
			}
		}
	}

	// Calculate travel fee based on service area
	travelFee := 0
	serviceAreas, err := mr.Store.ServiceAreas().GetByCleanerProfile(ctx, cleanerProfile.ID)
	if err == nil && len(serviceAreas) > 0 {
		// Get the address to match with service area
		address, _ := mr.Store.Addresses().Get(ctx, addressID)
		if address != nil {
			for _, area := range serviceAreas {
				if area.City == address.City {
					// Prefer exact postal code match
					if address.PostalCode != "" && area.PostalCode != "" && area.PostalCode == address.PostalCode {
						travelFee = area.TravelFee
						break
					}
					// Then neighborhood match
					if address.Neighborhood != "" && area.Neighborhood != "" && area.Neighborhood == address.Neighborhood {
						travelFee = area.TravelFee
						break
					}
					// Finally, just city match
					if area.PostalCode == "" && area.Neighborhood == "" {
						travelFee = area.TravelFee
					}
				}
			}
		}
	}

	// Calculate platform fee and totals
	// Platform fee is charged to customer, cleaner receives the service amount minus platform fee
	subtotal := servicePrice + addOnsPrice + travelFee
	platformFeePercentage := 15.0 // Default, could be from config
	platformFee := int(float64(subtotal) * platformFeePercentage / 100.0)
	totalPrice := subtotal + platformFee
	cleanerPayout := subtotal // Cleaner gets the subtotal (service + add-ons + travel)

	// Calculate total duration
	totalDuration := baseHours + addOnsDuration

	// Serialize service add-ons to JSON
	var addOnsJSON string
	if len(input.ServiceAddOns) > 0 {
		addOnsBytes, err := json.Marshal(input.ServiceAddOns)
		if err != nil {
			mr.Logger.Printf("Error marshaling service add-ons: %s", err)
			return nil, errors.New("error processing service add-ons")
		}
		addOnsJSON = string(addOnsBytes)
	}

	// Create booking
	customerNotes := ""
	if input.CustomerNotes != nil {
		customerNotes = *input.CustomerNotes
	}

	isRecurring := false
	if input.IsRecurring != nil {
		isRecurring = *input.IsRecurring
	}

	booking := &store.Booking{
		CustomerID:        userID,
		CleanerID:         cleanerProfile.UserID,
		CleanerProfileID:  cleanerProfile.ID,
		ServiceType:       input.ServiceType,
		ServiceFrequency:  input.ServiceFrequency,
		ServiceAddOns:     addOnsJSON,
		ScheduledDate:     input.ScheduledDate,
		ScheduledTime:     input.ScheduledTime,
		Duration:          totalDuration,
		AddressID:         addressID,
		CleanerHourlyRate: hourlyRate,
		ServicePrice:      servicePrice,
		AddOnsPrice:       addOnsPrice,
		TravelFee:         travelFee,
		PlatformFee:       platformFee,
		TotalPrice:        totalPrice,
		CleanerPayout:     cleanerPayout,
		Status:            store.BookingStatusPending,
		IsRecurring:       isRecurring,
		CustomerNotes:     customerNotes,
	}

	if err := mr.Store.Bookings().Create(ctx, booking); err != nil {
		mr.Logger.Printf("Error creating booking: %s", err)
		return nil, errors.New("error creating booking")
	}

	return booking, nil
}

func (mr *mutationResolver) UpdateBooking(ctx context.Context, input gen.UpdateBookingInput) (*store.Booking, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("authentication required")
	}

	// Get existing booking
	booking, err := mr.Store.Bookings().Get(ctx, input.ID)
	if err != nil {
		mr.Logger.Printf("Error retrieving booking: %s", err)
		return nil, errors.New("booking not found")
	}

	// Only customer can update booking details before confirmation
	if booking.CustomerID != currentUser.ID {
		return nil, errors.New("only the customer can update booking details")
	}

	// Can only update if booking is still pending
	if booking.Status != store.BookingStatusPending {
		return nil, errors.New("can only update pending bookings")
	}

	// Update fields
	if input.ScheduledDate != nil {
		booking.ScheduledDate = *input.ScheduledDate
	}
	if input.ScheduledTime != nil {
		booking.ScheduledTime = *input.ScheduledTime
	}
	if input.CustomerNotes != nil {
		booking.CustomerNotes = *input.CustomerNotes
	}
	if input.CleanerNotes != nil {
		booking.CleanerNotes = *input.CleanerNotes
	}

	if err := mr.Store.Bookings().Update(ctx, booking); err != nil {
		mr.Logger.Printf("Error updating booking: %s", err)
		return nil, errors.New("error updating booking")
	}

	return booking, nil
}

func (mr *mutationResolver) ConfirmBooking(ctx context.Context, id string) (*store.Booking, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("authentication required")
	}

	// Get booking
	booking, err := mr.Store.Bookings().Get(ctx, id)
	if err != nil {
		mr.Logger.Printf("Error retrieving booking: %s", err)
		return nil, errors.New("booking not found")
	}

	// Only cleaner assigned to this booking can confirm
	if booking.CleanerID != currentUser.ID {
		return nil, errors.New("only the assigned cleaner can confirm this booking")
	}

	// Can only confirm if booking is pending
	if booking.Status != store.BookingStatusPending {
		return nil, errors.New("can only confirm pending bookings")
	}

	// Update status
	booking.Status = store.BookingStatusConfirmed
	// Note: ConfirmedAt timestamp will be set by database trigger or we can set it here
	// For now, let's set it manually
	now := time.Now()
	booking.ConfirmedAt = &now

	if err := mr.Store.Bookings().Update(ctx, booking); err != nil {
		mr.Logger.Printf("Error confirming booking: %s", err)
		return nil, errors.New("error confirming booking")
	}

	return booking, nil
}

func (mr *mutationResolver) StartBooking(ctx context.Context, id string) (*store.Booking, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("authentication required")
	}

	// Get booking
	booking, err := mr.Store.Bookings().Get(ctx, id)
	if err != nil {
		mr.Logger.Printf("Error retrieving booking: %s", err)
		return nil, errors.New("booking not found")
	}

	// Only cleaner assigned to this booking can start it
	if booking.CleanerID != currentUser.ID {
		return nil, errors.New("only the assigned cleaner can start this booking")
	}

	// Can only start if booking is confirmed
	if booking.Status != store.BookingStatusConfirmed {
		return nil, errors.New("can only start confirmed bookings")
	}

	// Update status
	booking.Status = store.BookingStatusInProgress
	now := time.Now()
	booking.StartedAt = &now

	if err := mr.Store.Bookings().Update(ctx, booking); err != nil {
		mr.Logger.Printf("Error starting booking: %s", err)
		return nil, errors.New("error starting booking")
	}

	return booking, nil
}

func (mr *mutationResolver) CompleteBooking(ctx context.Context, id string, cleanerNotes *string) (*store.Booking, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("authentication required")
	}

	// Get booking
	booking, err := mr.Store.Bookings().Get(ctx, id)
	if err != nil {
		mr.Logger.Printf("Error retrieving booking: %s", err)
		return nil, errors.New("booking not found")
	}

	// Only cleaner assigned to this booking can complete it
	if booking.CleanerID != currentUser.ID {
		return nil, errors.New("only the assigned cleaner can complete this booking")
	}

	// Can only complete if booking is in progress
	if booking.Status != store.BookingStatusInProgress {
		return nil, errors.New("can only complete bookings that are in progress")
	}

	// Update status
	booking.Status = store.BookingStatusCompleted
	now := time.Now()
	booking.CompletedAt = &now

	if cleanerNotes != nil {
		booking.CleanerNotes = *cleanerNotes
	}

	if err := mr.Store.Bookings().Update(ctx, booking); err != nil {
		mr.Logger.Printf("Error completing booking: %s", err)
		return nil, errors.New("error completing booking")
	}

	// TODO: Trigger payout process
	// TODO: Send notification to customer to leave a review

	return booking, nil
}

func (mr *mutationResolver) CancelBooking(ctx context.Context, input gen.CancelBookingInput) (*store.Booking, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("authentication required")
	}

	// Get booking
	booking, err := mr.Store.Bookings().Get(ctx, input.ID)
	if err != nil {
		mr.Logger.Printf("Error retrieving booking: %s", err)
		return nil, errors.New("booking not found")
	}

	// Customer or cleaner can cancel
	if booking.CustomerID != currentUser.ID && booking.CleanerID != currentUser.ID && !currentUser.IsGlobalAdmin() {
		return nil, errors.New("only customer, cleaner, or admin can cancel this booking")
	}

	// Can't cancel if already completed or cancelled
	if booking.Status == store.BookingStatusCompleted || booking.Status == store.BookingStatusCancelled {
		return nil, errors.New("cannot cancel completed or already cancelled bookings")
	}

	// Update status
	booking.Status = store.BookingStatusCancelled
	booking.CancellationReason = (*store.CancellationReason)(&input.Reason)
	if input.Note != nil {
		booking.CancellationNote = *input.Note
	}
	now := time.Now()
	booking.CancelledAt = &now
	booking.CancelledByID = &currentUser.ID

	if err := mr.Store.Bookings().Update(ctx, booking); err != nil {
		mr.Logger.Printf("Error cancelling booking: %s", err)
		return nil, errors.New("error cancelling booking")
	}

	// TODO: Implement refund logic based on cancellation policy
	// Full refund if cancelled 24+ hours before scheduled date

	return booking, nil
}

func (mr *mutationResolver) MarkNoShow(ctx context.Context, id string) (*store.Booking, error) {
	currentUser := middleware.GetCurrentUser(ctx)
	if currentUser == nil {
		return nil, errors.New("authentication required")
	}

	// Get booking
	booking, err := mr.Store.Bookings().Get(ctx, id)
	if err != nil {
		mr.Logger.Printf("Error retrieving booking: %s", err)
		return nil, errors.New("booking not found")
	}

	// Only cleaner assigned to this booking can mark no-show
	if booking.CleanerID != currentUser.ID {
		return nil, errors.New("only the assigned cleaner can mark customer as no-show")
	}

	// Can only mark no-show if booking is confirmed
	if booking.Status != store.BookingStatusConfirmed {
		return nil, errors.New("can only mark no-show for confirmed bookings")
	}

	// Update status
	booking.Status = store.BookingStatusNoShow

	if err := mr.Store.Bookings().Update(ctx, booking); err != nil {
		mr.Logger.Printf("Error marking booking as no-show: %s", err)
		return nil, errors.New("error marking booking as no-show")
	}

	// TODO: Apply no-show fee to customer
	// TODO: Compensate cleaner for travel

	return booking, nil
}
