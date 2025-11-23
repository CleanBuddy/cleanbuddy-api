package postgresql

import (
	"fmt"
	"runtime"

	"saas-starter-api/res/store"

	sqlCommenter "github.com/gouyelliot/gorm-sqlcommenter-plugin"
	"github.com/graph-gophers/dataloader"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type storeImpl struct {
	db *gorm.DB

	authSessionStore *authSessionStore
	userStore        *userStore
	applicationStore *applicationStore
}

func (sImpl *storeImpl) AuthSessions() store.AuthSessionStore {
	return sImpl.authSessionStore
}

func (sImpl *storeImpl) Users() store.UserStore {
	return sImpl.userStore
}

func (sImpl *storeImpl) Applications() store.ApplicationStore {
	return sImpl.applicationStore
}

func (sImpl *storeImpl) GetDB() interface{} {
	return sImpl.db
}

func Connect(connectionUrl string) (*storeImpl, error) {
	db, err := gorm.Open(postgres.Open(connectionUrl), &gorm.Config{TranslateError: true, PrepareStmt: false})
	if err != nil {
		return nil, err
	}

	err = db.Use(sqlCommenter.New())
	if err != nil {
		return nil, err
	}

	err = decorateDBOperationsWithAdditionalInfo(db)
	if err != nil {
		return nil, err
	}

	s := &storeImpl{db: db}

	s.authSessionStore = NewAuthSessionStore(s)
	s.userStore = NewUserStore(s)
	s.applicationStore = NewApplicationStore(s)

	return s, nil
}

// COMMON UTILITIES

func decorateBatchedQueriesWithError(err error, keys []dataloader.Key) []*dataloader.Result {
	var results []*dataloader.Result

	for i := 0; i < len(keys); i++ {
		results = append(results, &dataloader.Result{Data: nil, Error: err})
	}

	return results
}

func identifyCallee(stackDepth int) string {
	function, _, line, ok := runtime.Caller(stackDepth)
	if !ok {
		return "<missing-runtime-info>"
	}
	return fmt.Sprintf("%s:%d", runtime.FuncForPC(function).Name(), line)
}

func annotateWithInfoHook(db *gorm.DB) {
	info := identifyCallee(4) // Skip the internal gorm calls & the 2 local setup calls
	db.Clauses(sqlCommenter.NewTag("action", info))
}

func decorateDBOperationsWithAdditionalInfo(db *gorm.DB) error {
	return db.Callback().Query().Before("gorm:query").Register("store::annotate_with_info", annotateWithInfoHook)
}
