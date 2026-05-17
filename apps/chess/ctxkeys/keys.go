package ctxkeys

type storeKeyType struct{}
type gameRepoKeyType struct{}
type dbRepoKeyType struct{}
type analysisRunnerKeyType struct{}

var (
	StoreKey          = storeKeyType{}
	GameRepoKey       = gameRepoKeyType{}
	DBRepoKey         = dbRepoKeyType{}
	AnalysisRunnerKey = analysisRunnerKeyType{}
)
