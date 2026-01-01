package ctxkeys

type storeKeyType struct{}
type gameRepoKeyType struct{}

var (
	StoreKey    = storeKeyType{}
	GameRepoKey = gameRepoKeyType{}
)
