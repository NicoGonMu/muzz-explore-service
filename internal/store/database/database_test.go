package database

import (
	"context"
	"database/sql"
	"muzz-explore/internal/store"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func ref[T any](t T) *T { return &t }

type dbTestSuite struct {
	suite.Suite
	db       *sql.DB
	dbCloser func() error
	mock     sqlmock.Sqlmock
}

func TestIntegration(t *testing.T) {
	suite.Run(t, &dbTestSuite{})
}

func (s *dbTestSuite) BeforeTest(_, testName string) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(s.T(), err)
	s.db = db
	s.mock = mock
}

func (s *dbTestSuite) AfterTest(_, testName string) {
}

func (s *dbTestSuite) Test_ListDecisionsNoFilters() {
	// GIVEN database set up with some expectations.
	s.mock.ExpectQuery("SELECT * FROM decisions ORDER BY actor_user_id,recipient_user_id LIMIT 10").WillReturnRows(
		s.mock.NewRows(
			[]string{"actor_user_id", "recipient_user_id", "liked_recipient", "last_modified", "seen_by_recipient"},
		).AddRow("actor", "recipient", true, 123, false),
	)
	db := database{db: s.db}

	// WHEN ListDecisions is called with no filters.
	got, gotPage, err := db.ListDecisions(context.Background(), store.DecisionFilter{}, "")

	// THEN the expectations are met and the result is as expected.
	require.NoError(s.T(), err)
	assert.Equal(s.T(), []store.Decision{
		{
			ActorUserID:     "actor",
			RecipientUserID: "recipient",
			LikedRecipient:  true,
			LastModified:    123,
			SeenByRecipient: false,
		},
	}, got)
	assert.Equal(s.T(), "actor##recipient", gotPage)
	assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}

func (s *dbTestSuite) Test_ListDecisionsWithPage() {
	// GIVEN database set up with some expectations.
	s.mock.ExpectQuery("SELECT * FROM decisions WHERE actor_user_id>? AND recipient_user_id>? ORDER BY actor_user_id,recipient_user_id LIMIT 10").
		WithArgs("actor", "recipient").
		WillReturnRows(
			s.mock.NewRows(
				[]string{"actor_user_id", "recipient_user_id", "liked_recipient", "last_modified", "seen_by_recipient"},
			).AddRow("actor", "recipient", true, 123, false),
		)
	db := database{db: s.db}

	// WHEN ListDecisions is called with no filters.
	got, gotPage, err := db.ListDecisions(context.Background(), store.DecisionFilter{}, "actor##recipient")

	// THEN the expectations are met and the result is as expected.
	require.NoError(s.T(), err)
	assert.Equal(s.T(), []store.Decision{
		{
			ActorUserID:     "actor",
			RecipientUserID: "recipient",
			LikedRecipient:  true,
			LastModified:    123,
			SeenByRecipient: false,
		},
	}, got)
	assert.Equal(s.T(), "actor##recipient", gotPage)
	assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}

func (s *dbTestSuite) Test_ListDecisionsAllFilters() {
	// GIVEN database set up with some expectations.
	s.mock.ExpectQuery(
		"SELECT * FROM decisions WHERE actor_user_id=? AND recipient_user_id=? AND liked_recipient=? AND last_modified=? AND seen_by_recipient=? ORDER BY actor_user_id,recipient_user_id LIMIT 10").
		WillReturnRows(
			s.mock.NewRows(
				[]string{"actor_user_id", "recipient_user_id", "seen_by_recipient", "seen_by_recipient", "seen_by_recipient"},
			).AddRow("actor", "recipient", true, 123, false),
		)
	db := database{db: s.db}

	// WHEN ListDecisions is called with filters.
	got, gotPage, err := db.ListDecisions(context.Background(), store.DecisionFilter{
		ActorUserID:     ref("actor"),
		RecipientUserID: ref("recipient"),
		LikedRecipient:  ref(true),
		LastModified:    ref(uint64(123)),
		SeenByRecipient: ref(false),
	}, "")

	// THEN the expectations are met and the result is as expected.
	require.NoError(s.T(), err)
	assert.Equal(s.T(), []store.Decision{
		{
			ActorUserID:     "actor",
			RecipientUserID: "recipient",
			LikedRecipient:  true,
			LastModified:    123,
			SeenByRecipient: false,
		},
	}, got)
	assert.Equal(s.T(), "actor##recipient", gotPage)
	assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}

func (s *dbTestSuite) Test_CountDecisionsNoFilters() {
	// GIVEN database set up with some expectations.
	s.mock.ExpectQuery("SELECT COUNT(*) FROM decisions").WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(3),
	)
	db := database{db: s.db}
	got, err := db.CountDecisions(context.Background(), store.DecisionFilter{})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), uint64(3), got)
	assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}

func (s *dbTestSuite) Test_CountDecisionsAllFilters() {
	// GIVEN database set up with some expectations.
	s.mock.ExpectQuery("SELECT COUNT(*) FROM decisions WHERE actor_user_id=? AND recipient_user_id=? AND liked_recipient=? AND last_modified=? AND seen_by_recipient=?").
		WithArgs("actor", "recipient", true, 123, false).WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(3),
	)
	db := database{db: s.db}

	// WHEN CountDecisions is called with all the filters.
	got, err := db.CountDecisions(context.Background(), store.DecisionFilter{
		ActorUserID:     ref("actor"),
		RecipientUserID: ref("recipient"),
		LikedRecipient:  ref(true),
		LastModified:    ref(uint64(123)),
		SeenByRecipient: ref(false),
	})

	// THEN the expectations are met and the result is as expected.
	require.NoError(s.T(), err)
	assert.Equal(s.T(), uint64(3), got)
	assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}

func (s *dbTestSuite) Test_UpsertDecision() {
	// GIVEN database set up with some expectations.
	s.mock.ExpectExec("REPLACE INTO decisions (actor_user_id,recipient_user_id,liked_recipient,last_modified,seen_by_recipient) VALUES (?,?,?,?,?)").
		WithArgs("actor", "recipient", true, 123, false).WillReturnResult(sqlmock.NewResult(1, 1))
	db := database{db: s.db}

	// WHEN UpsertDecision is called.
	err := db.UpsertDecision(context.Background(), store.Decision{
		ActorUserID:     "actor",
		RecipientUserID: "recipient",
		LikedRecipient:  true,
		LastModified:    123,
		SeenByRecipient: false,
	})

	// THEN the expectations are met and the result is as expected.
	require.NoError(s.T(), err)
	assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}

func (s *dbTestSuite) Test_MarkDecisionsAsSeenNoInitPage() {
	// GIVEN database set up with some expectations.
	s.mock.ExpectExec("UPDATE decisions SET seen_by_recipient = ? WHERE recipient_user_id=? AND actor_user_id<=?").
		WithArgs(true, "recipient", "actor20").WillReturnResult(sqlmock.NewResult(1, 1))
	db := database{db: s.db}

	// WHEN MarkDecisionsAsSeen is called.
	err := db.MarkDecisionsAsSeen(context.Background(), "recipient", "", "actor20##recipient")

	// THEN the expectations are met and the result is as expected.
	require.NoError(s.T(), err)
	assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}

func (s *dbTestSuite) Test_MarkDecisionsAsSeenInitPage() {
	// GIVEN database set up with some expectations.
	s.mock.ExpectExec("UPDATE decisions SET seen_by_recipient = ? WHERE recipient_user_id=? AND actor_user_id<=? AND actor_user_id>?").
		WithArgs(true, "recipient", "actor20", "actor10").WillReturnResult(sqlmock.NewResult(1, 1))
	db := database{db: s.db}

	// WHEN MarkDecisionsAsSeen is called.
	err := db.MarkDecisionsAsSeen(context.Background(), "recipient", "actor10##recipient", "actor20##recipient")

	// THEN the expectations are met and the result is as expected.
	require.NoError(s.T(), err)
	assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}
