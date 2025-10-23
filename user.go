package ncservice

type AuthorizedUser interface {
	GetUserId() string
}

type AuthorizedService interface {
	AuthorizedUser
	IsAdmin() bool
}

type AuthorizedCustomer interface {
	AuthorizedUser
	GetCustomerId() int64
	GetAccountId() int64
}
