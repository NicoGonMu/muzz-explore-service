// This file contains the database implementation, oriented to MySQL.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"muzz-explore/internal/store"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog/log"
)

const PageLength = 10

type database struct {
	db *sql.DB
}

func NewClient(user, pass, address, port, dbname string) (*database, func() error, error) {
	// Create a new connection to the database.
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, pass, address, port, dbname)
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}
	return &database{db}, db.Close, nil
}

func (d *database) ListDecisions(
	ctx context.Context,
	filter store.DecisionFilter,
	page string,
) ([]store.Decision, string, error) {
	sb := addFilters(sq.Select("*").From("decisions"), filter)
	if page != "" {
		pageIDs := strings.Split(page, "##")
		sb = sb.Where("actor_user_id>?", pageIDs[0])
		sb = sb.Where("recipient_user_id>?", pageIDs[1])
	}
	sb = sb.OrderBy("actor_user_id,recipient_user_id").Limit(PageLength)
	results, err := sb.RunWith(d.db).QueryContext(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list decisions: %w", err)
	}
	decisions := []store.Decision{}
	for results.Next() {
		var decision store.Decision
		if err := results.Scan(
			&decision.ActorUserID,
			&decision.RecipientUserID,
			&decision.LikedRecipient,
			&decision.LastModified,
			&decision.SeenByRecipient,
		); err != nil {
			// Log error and continue to next row, as we don't want to loose the whole query.
			log.Err(err).Msg("failed to scan decision")
			continue
		}
		decisions = append(decisions, store.Decision(decision))
	}
	numDecisions := len(decisions)
	if numDecisions == 0 {
		return decisions, "", nil
	}
	lastDecision := decisions[numDecisions-1]
	return decisions, fmt.Sprintf("%s##%s", lastDecision.ActorUserID, lastDecision.RecipientUserID), nil
}

func (d *database) CountDecisions(ctx context.Context, filter store.DecisionFilter) (uint64, error) {
	sb := addFilters(sq.Select("COUNT(*)").From("decisions"), filter)
	var count uint64
	err := sb.RunWith(d.db).QueryRowContext(ctx).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count decisions: %w", err)
	}
	return count, nil
}

func (d *database) UpsertDecision(ctx context.Context, decision store.Decision) error {
	_, err := sq.Replace("decisions").Columns(
		"actor_user_id",
		"recipient_user_id",
		"liked_recipient",
		"last_modified",
		"seen_by_recipient",
	).Values(
		decision.ActorUserID,
		decision.RecipientUserID,
		decision.LikedRecipient,
		decision.LastModified,
		decision.SeenByRecipient,
	).RunWith(d.db).ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to upsert decision: %w", err)
	}
	return nil
}

func (d *database) MarkDecisionsAsSeen(ctx context.Context, recipientUserID string, timestamp int64) error {
	_, err := sq.Update("decisions").Set("seen_by_recipient", true).
		Where("recipient_user_id=?", recipientUserID).
		// Intentionally not using <= to avoid timing issues; better to show a new twice than not showing it at all.
		Where("last_modified<?", timestamp).
		RunWith(d.db).ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to update decisions: %w", err)
	}
	return nil
}

func addFilters(sb sq.SelectBuilder, filter store.DecisionFilter) sq.SelectBuilder {
	if filter.ActorUserID != nil {
		sb = sb.Where("actor_user_id=?", *filter.ActorUserID)
	}
	if filter.RecipientUserID != nil {
		sb = sb.Where("recipient_user_id=?", *filter.RecipientUserID)
	}
	if filter.LikedRecipient != nil {
		sb = sb.Where("liked_recipient=?", *filter.LikedRecipient)
	}
	if filter.LastModified != nil {
		sb = sb.Where("last_modified=?", *filter.LastModified)
	}
	if filter.SeenByRecipient != nil {
		sb = sb.Where("seen_by_recipient=?", *filter.SeenByRecipient)
	}

	return sb
}
