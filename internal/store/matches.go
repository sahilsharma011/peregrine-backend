package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Match holds information about an FRC match at a specific event
type Match struct {
	Key                string         `json:"key" db:"key"`
	EventKey           string         `json:"eventKey" db:"event_key"`
	PredictedTime      *time.Time     `json:"predictedTime" db:"predicted_time"`
	ActualTime         *time.Time     `json:"actualTime" db:"actual_time"`
	ScheduledTime      *time.Time     `json:"scheduledTime" db:"scheduled_time"`
	RedScore           *int           `json:"redScore" db:"red_score"`
	BlueScore          *int           `json:"blueScore" db:"blue_score"`
	RedAlliance        pq.StringArray `json:"redAlliance" db:"red_alliance"`
	BlueAlliance       pq.StringArray `json:"blueAlliance" db:"blue_alliance"`
	TBADeleted         bool           `json:"tbaDeleted" db:"tba_deleted"`
	RedScoreBreakdown  ScoreBreakdown `json:"redScoreBreakdown" db:"red_score_breakdown"`
	BlueScoreBreakdown ScoreBreakdown `json:"blueScoreBreakdown" db:"blue_score_breakdown"`
	TBAURL             *string        `json:"tbaUrl" db:"tba_url"`
}

// ScoreBreakdown changes year to year, but it's generally a map of strings
// to strings, integers, or booleans.
type ScoreBreakdown map[string]interface{}

// Value returns the JSON representation of the score breakdown. Since this
// changes year to year we just store it as arbitrary JSON.
func (sb ScoreBreakdown) Value() (driver.Value, error) {
	return json.Marshal(sb)
}

// Scan unmarshals the JSON representation of the score breakdown stored in
// the database into the score breakdown.
func (sb ScoreBreakdown) Scan(src interface{}) error {
	j, ok := src.([]byte)
	if !ok {
		return errors.New("got invalid type for ScoreBreakdown")
	}

	return json.Unmarshal(j, &sb)
}

// GetTime returns the actual match time if available, and if not, predicted time
func (m *Match) GetTime() *time.Time {
	if m.ActualTime != nil {
		return m.ActualTime
	}
	if m.PredictedTime != nil {
		return m.PredictedTime
	}
	return m.ScheduledTime
}

const matchesQuery = `
SELECT
	matches.key,
	matches.predicted_time,
	matches.scheduled_time,
	matches.actual_time,
	matches.blue_score,
	matches.red_score,
	matches.tba_deleted,
	r.team_keys AS red_alliance,
	b.team_keys AS blue_alliance,
	matches.red_score_breakdown,
	matches.blue_score_breakdown,
	matches.tba_url
FROM
	matches
INNER JOIN
	alliances r
	ON
		matches.key = r.match_key AND r.is_blue = false
INNER JOIN
	alliances b
	ON
		matches.key = b.match_key AND b.is_blue = true
INNER JOIN
	events
	ON
		matches.event_key = events.key
WHERE
	(events.realm_id = $1 OR events.realm_id IS NULL)`

// GetMatchesForRealm returns all matches for a realm from a specific event that include the given
// teams. If teams is nil or empty a list of all the matches for that event are
// returned. // If tbaDeleted is true, matches that have been deleted from TBA
// will be returned in addition to matches that have not been deleted. Otherwise,
// only matches that have not been deleted will be returned.
func (s *Service) GetMatchesForRealm(ctx context.Context, eventKey string, teamKeys []string, tbaDeleted bool, realmID *int64) ([]Match, error) {
	if teamKeys == nil {
		teamKeys = []string{}
	}

	query := matchesQuery + " AND matches.event_key = $2 AND (r.team_keys || b.team_keys) @> $3"
	if !tbaDeleted {
		query += " AND NOT matches.tba_deleted"
	}

	matches := make([]Match, 0)
	err := s.db.SelectContext(ctx, &matches, query, realmID, eventKey, pq.Array(teamKeys))
	if err != nil {
		return nil, err
	}

	return matches, nil
}

// GetEventRealmIDByMatchKeyTx returns the realm ID for the event that the match associated
// identified by the given key is associated with.
func (s *Service) GetEventRealmIDByMatchKeyTx(ctx context.Context, tx *sqlx.Tx, matchKey string) (realmID *int64, err error) {
	err = tx.QueryRowContext(ctx, `
	SELECT events.realm_id
	FROM matches
	LEFT JOIN events
		ON events.key = matches.event_key
	WHERE matches.key = $1
	`, matchKey).Scan(&realmID)
	if err == sql.ErrNoRows {
		return nil, ErrNoResults{fmt.Errorf("couldn't find match by key: %w", err)}
	} else if err != nil {
		return realmID, fmt.Errorf("unable to determine event realm ID for match: %w", err)
	}

	return realmID, nil
}

// GetMatchForRealm returns a specific match by key in the given realm.
func (s *Service) GetMatchForRealm(ctx context.Context, matchKey string, realmID *int64) (Match, error) {
	const query = matchesQuery + " AND matches.key = $2"

	var m Match
	err := s.db.GetContext(ctx, &m, query, realmID, matchKey)
	if err == sql.ErrNoRows {
		return m, ErrNoResults{fmt.Errorf("unable to get match: %w", err)}
	} else if err != nil {
		return m, fmt.Errorf("unable to get match: %w", err)
	}

	return m, nil
}

// ExclusiveLockMatchesTx locks the matches table so no changes can be made to it by anything other
// than the given transaction.
func (s *Service) ExclusiveLockMatchesTx(ctx context.Context, tx *sqlx.Tx) error {
	_, err := tx.ExecContext(ctx, "LOCK TABLE matches IN EXCLUSIVE MODE")
	if err != nil {
		return fmt.Errorf("unable to lock matches: %w", err)
	}

	return nil
}

// DeleteMatchTx deletes a specific match using the given transaction.
func (s *Service) DeleteMatchTx(ctx context.Context, tx *sqlx.Tx, matchKey string) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM matches WHERE key = $1`, matchKey)
	if err != nil {
		return fmt.Errorf("unable to delete match: %w", err)
	}

	return nil
}

// UpsertMatchTx upserts a match and its alliances into the database in the given transaction.
func (s *Service) UpsertMatchTx(ctx context.Context, tx *sqlx.Tx, match Match) error {
	_, err := tx.NamedExecContext(ctx, `
		INSERT INTO matches (key, event_key, predicted_time, scheduled_time, actual_time, red_score, blue_score, tba_deleted, red_score_breakdown, blue_score_breakdown, tba_url)
		VALUES (:key, :event_key, :predicted_time, :scheduled_time, :actual_time, :red_score, :blue_score, :tba_deleted, :red_score_breakdown, :blue_score_breakdown, :tba_url)
		ON CONFLICT (key)
		DO
			UPDATE
				SET
					event_key = :event_key,
					predicted_time = :predicted_time,
					scheduled_time = :scheduled_time,
					actual_time = :actual_time,
					red_score = :red_score,
					blue_score = :blue_score,
					tba_deleted = :tba_deleted,
					red_score_breakdown = :red_score_breakdown,
					blue_score_breakdown = :blue_score_breakdown,
					tba_url = :tba_url
		`, match)
	if err != nil {
		return fmt.Errorf("unable to upsert matches: %w", err)
	}

	if err = s.AlliancesUpsertTx(ctx, tx, match.Key, match.BlueAlliance, match.RedAlliance); err != nil {
		return fmt.Errorf("unable to upsert alliances: %w", err)
	}

	if err := s.EventTeamKeysUpsertTx(ctx, tx, match.EventKey, append(match.BlueAlliance, match.RedAlliance...)); err != nil {
		return fmt.Errorf("unable to upsert event team keys: %w", err)
	}

	return nil
}

// MarkMatchesDeleted will set tba_deleted to true on all matches for an event
// that were *not* included in the passed matches slice.
func (s *Service) MarkMatchesDeleted(ctx context.Context, eventKey string, matches []Match) error {
	keys := pq.StringArray{}
	for _, e := range matches {
		keys = append(keys, e.Key)
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE matches
			SET
				tba_deleted = true
			WHERE
				event_key = $1 AND
				key != ALL($2)
	`, eventKey, keys)
	if err != nil {
		return fmt.Errorf("unable to mark tba_deleted on missing matches: %w", err)
	}

	return nil
}

// UpdateTBAMatches puts a set of multiple matches and their alliances from TBA
// into the database. New matches are added, existing matches will be updated,
// and matches deleted from TBA will be deleted from the database. User-created
// matches will be unaffected. If eventKey is specified, only matches from that
// event will be affected. It will set tba_deleted to false for all updated matches.
func (s *Service) UpdateTBAMatches(ctx context.Context, eventKey string, matches []Match) error {
	return s.DoTransaction(ctx, func(tx *sqlx.Tx) error {
		upsert, err := tx.PrepareNamedContext(ctx, `
		INSERT INTO matches (key, event_key, predicted_time, scheduled_time, actual_time, red_score, blue_score, tba_deleted, red_score_breakdown, blue_score_breakdown, tba_url)
		VALUES (:key, :event_key, :predicted_time, :scheduled_time, :actual_time, :red_score, :blue_score, :tba_deleted, :red_score_breakdown, :blue_score_breakdown, :tba_url)
		ON CONFLICT (key)
		DO
			UPDATE
				SET
					event_key = :event_key,
					predicted_time = :predicted_time,
					scheduled_time = :scheduled_time,
					actual_time = :actual_time,
					red_score = :red_score,
					blue_score = :blue_score,
					tba_deleted = false,
					red_score_breakdown = :red_score_breakdown,
					blue_score_breakdown = :blue_score_breakdown,
					tba_url = :tba_url
	`)
		if err != nil {
			return fmt.Errorf("unable to prepare query to upsert matches: %w", err)
		}

		for _, match := range matches {
			if _, err = upsert.ExecContext(ctx, match); err != nil {
				return fmt.Errorf("unable to upsert match: %w", err)
			}

			if err = s.AlliancesUpsertTx(ctx, tx, match.Key, match.BlueAlliance, match.RedAlliance); err != nil {
				return fmt.Errorf("unable to upsert alliances: %w", err)
			}
		}

		return nil
	})
}

const analysisInfoQuery = `
SELECT
	matches.key,
	r.team_keys AS red_alliance,
	b.team_keys AS blue_alliance,
	matches.red_score_breakdown,
	matches.blue_score_breakdown
FROM
	matches
INNER JOIN
	alliances r
ON
	matches.key = r.match_key AND r.is_blue = false
INNER JOIN
	alliances b
ON
	matches.key = b.match_key AND b.is_blue = true
INNER JOIN
	events
	ON
		matches.event_key = events.key	
WHERE
	(events.realm_id = $1 OR events.realm_id IS NULL) AND
	matches.event_key = $2`

// GetEventAnalysisInfoForRealm returns match information that's pertinent to doing analysis by getting
// all the matches with the given event key and either null or matching realm IDs.
func (s *Service) GetEventAnalysisInfoForRealm(ctx context.Context, eventKey string, realmID *int64) ([]Match, error) {
	matches := make([]Match, 0)

	err := s.db.SelectContext(ctx, &matches, analysisInfoQuery, realmID, eventKey)
	if err != nil {
		return matches, fmt.Errorf("unable to get analysis info: %w", err)
	}

	return matches, nil
}

// GetMatchAnalysisInfoForRealm returns match information that's pertinent to doing analysis by getting
// all the matches with the given event key and either null or matching realm IDs.
func (s *Service) GetMatchAnalysisInfoForRealm(ctx context.Context, eventKey, matchKey string, realmID *int64) (match Match, err error) {
	const query = analysisInfoQuery + "AND matches.key = $3"

	err = s.db.GetContext(ctx, &match, query, realmID, eventKey, matchKey)
	if err == sql.ErrNoRows {
		return match, ErrNoResults{fmt.Errorf("unable to find match: %w", err)}
	} else if err != nil {
		return match, fmt.Errorf("unable to get analysis info: %w", err)
	}

	return match, nil
}
