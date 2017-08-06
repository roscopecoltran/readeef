package db

import "github.com/jmoiron/sqlx"

type Helper interface {
	SQL() SqlStmts
	InitSQL() []string

	CreateWithID(tx *sqlx.Tx, sql string, args ...interface{}) (int64, error)
	Upgrade(db *DB, old, new int) error

	WhereMultipleORs(string, int, int) string
}

type ArticleStmts struct {
	Create string
	Update string

	CreateUserUnread   string
	DeleteUserUnread   string
	CreateUserFavorite string
	DeleteUserFavorite string

	GetScores    string
	CreateScores string
	UpdateScores string

	GetThumbnail    string
	CreateThumbnail string
	UpdateThumbnail string

	GetExtract    string
	CreateExtract string
	UpdateExtract string

	GetArticlesTemplate string
}

type FeedStmts struct {
	IDs    string
	Create string
	Update string
	Delete string

	GetAllArticles    string
	GetLatestArticles string

	GetHubbubSubscription   string
	GetUsers                string
	Attach                  string
	Detach                  string
	GetUserTags             string
	CreateUserTag           string
	DeleteUserTags          string
	ReadStateInsertTemplate string
}

type RepoStmts struct {
	GetUser                  string
	GetUserByMD5API          string
	GetUsers                 string
	GetFeed                  string
	GetFeedByLink            string
	GetFeeds                 string
	GetUnsubscribedFeeds     string
	GetHubbubSubscriptions   string
	FailHubbubSubscriptions  string
	DeleteStaleUnreadRecords string
}

type SubscriptionStmts struct {
	Create string
	Update string
	Delete string
}

type TagStmts struct {
	GetByValue     string
	Create         string
	Update         string
	GetUserFeeds   string
	GetUserFeedIDs string
	DeleteStale    string

	GetArticlesJoin     string
	ReadStateInsertJoin string
	ReadStateDeleteJoin string
	ArticleCountJoin    string
}

type UserStmts struct {
	Create string
	Update string
	Delete string

	GetFeed        string
	GetFeeds       string
	GetFeedIdsTags string

	GetTags       string
	GetTag        string
	GetTagByValue string

	GetArticlesUserlessTemplate string
	GetArticlesTemplate         string
	GetArticlesScoreJoin        string
	GetArticlesUntaggedJoin     string

	GetArticleIdsTemplate      string
	GetArticleIdsUserFeedsJoin string
	GetArticleIdsUnreadJoin    string
	GetArticleIdsFavoriteJoin  string
	GetArticleIdsUntaggedJoin  string

	StateUnreadJoin             string
	StateFavoriteJoin           string
	ReadStateInsertTemplate     string
	ReadStateDeleteTemplate     string
	FavoriteStateInsertTemplate string
	FavoriteStateDeleteTemplate string

	ArticleCountTemplate      string
	ArticleCountUserFeedsJoin string
	ArticleCountUnreadJoin    string
	ArticleCountFavoriteJoin  string
	ArticleCountUntaggedJoin  string
}

type SqlStmts struct {
	Article      ArticleStmts
	Feed         FeedStmts
	Repo         RepoStmts
	Subscription SubscriptionStmts
	Tag          TagStmts
	User         UserStmts
}

func Register(driver string, helper Helper) {
	if helper == nil {
		panic("No helper provided")
	}

	if _, ok := helpers[driver]; ok {
		panic("Helper " + driver + " already registered")
	}

	helpers[driver] = helper
}

// Can't recover from missing driver or statement, panic
func (db DB) SQL() SqlStmts {
	driver := db.DriverName()

	if h, ok := helpers[driver]; ok {
		return h.SQL()
	} else {
		panic("No helper registered for " + driver)
	}
}
