package ctxkeys

type storeKeyType struct{}
type gameRepoKeyType struct{}
type dbRepoKeyType struct{}

var (
	StoreKey    = storeKeyType{}
	GameRepoKey = gameRepoKeyType{}
	DBRepoKey   = dbRepoKeyType{}
)
