package postgresql

import (
	"fmt"
	"runtime"

	"cleanbuddy-api/res/store"

	sqlCommenter "github.com/gouyelliot/gorm-sqlcommenter-plugin"
	"github.com/graph-gophers/dataloader"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type storeImpl struct {
	db *gorm.DB

	authSessionStore    *authSessionStore
	userStore           *userStore
	applicationStore    *applicationStore
	cleanerProfileStore *cleanerProfileStore
	serviceAreaStore    *serviceAreaStore
	addressStore        *addressStore
	serviceStore        *serviceStore
	bookingStore        *bookingStore
	reviewStore         *reviewStore
	transactionStore    *transactionStore
	availabilityStore   *availabilityStore
	companyStore        *companyStore
	cleanerInviteStore  *cleanerInviteStore
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

func (sImpl *storeImpl) CleanerProfiles() store.CleanerProfileStore {
	return sImpl.cleanerProfileStore
}

func (sImpl *storeImpl) ServiceAreas() store.ServiceAreaStore {
	return sImpl.serviceAreaStore
}

func (sImpl *storeImpl) Addresses() store.AddressStore {
	return sImpl.addressStore
}

func (sImpl *storeImpl) Services() store.ServiceStore {
	return sImpl.serviceStore
}

func (sImpl *storeImpl) Bookings() store.BookingStore {
	return sImpl.bookingStore
}

func (sImpl *storeImpl) Reviews() store.ReviewStore {
	return sImpl.reviewStore
}

func (sImpl *storeImpl) Transactions() store.TransactionStore {
	return sImpl.transactionStore
}

func (sImpl *storeImpl) Availability() store.AvailabilityStore {
	return sImpl.availabilityStore
}

func (sImpl *storeImpl) Companies() store.CompanyStore {
	return sImpl.companyStore
}

func (sImpl *storeImpl) CleanerInvites() store.CleanerInviteStore {
	return sImpl.cleanerInviteStore
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

	// Auto-migrate all tables
	// err = db.AutoMigrate(
	// 	&store.User{},
	// 	&store.AuthSession{},
	// 	&store.Application{},
	// 	&store.Company{},
	// 	&store.CleanerProfile{},
	// 	&store.CleanerInvite{},
	// 	&store.ServiceArea{},
	// 	&store.Address{},
	// 	&store.ServiceDefinition{},
	// 	&store.ServiceAddOnDefinition{},
	// 	&store.Booking{},
	// 	&store.Review{},
	// 	&store.Transaction{},
	// 	&store.PayoutBatch{},
	// 	&store.Availability{},
	// )
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to auto-migrate tables: %w", err)
	// }

	s := &storeImpl{db: db}

	s.authSessionStore = NewAuthSessionStore(s)
	s.userStore = NewUserStore(s)
	s.applicationStore = NewApplicationStore(s)
	s.cleanerProfileStore = NewCleanerProfileStore(s)
	s.serviceAreaStore = NewServiceAreaStore(s)
	s.addressStore = NewAddressStore(s)
	s.serviceStore = NewServiceStore(s)
	s.bookingStore = NewBookingStore(s)
	s.reviewStore = NewReviewStore(s)
	s.transactionStore = NewTransactionStore(s)
	s.availabilityStore = NewAvailabilityStore(s)
	s.companyStore = NewCompanyStore(s)
	s.cleanerInviteStore = NewCleanerInviteStore(s)

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
