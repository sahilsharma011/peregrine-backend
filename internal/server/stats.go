package server

import (
	"fmt"
	"net/http"

	ihttp "github.com/Pigmice2733/peregrine-backend/internal/http"
	"github.com/Pigmice2733/peregrine-backend/internal/store"
	"github.com/Pigmice2733/peregrine-backend/internal/summary"
	"github.com/gorilla/mux"
)

// eventStats analyzes the event-wide statistics of every team at an event with submitted reports
func (s *Server) eventStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]

		var realmID *int64
		userRealmID, realmErr := ihttp.GetRealmID(r)
		if realmErr != nil {
			realmID = &userRealmID
		}

		event, err := s.Store.GetEventForRealm(r.Context(), eventKey, realmID)
		if _, ok := err.(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving event")
			return
		}

		// TODO: handle this better by crafting a better query
		// if !s.checkEventAccess(event.RealmID, r) {
		// 	ihttp.Error(w, http.StatusForbidden)
		// 	return
		// }

		if event.SchemaID == nil {
			ihttp.Respond(w, fmt.Errorf("no schema found"), http.StatusBadRequest)
			return
		}

		var reports []store.Report

		if ihttp.GetRoles(r).IsSuperAdmin {
			reports, err = s.Store.GetEventReports(r.Context(), eventKey)
		} else {
			reports, err = s.Store.GetEventReportsForRealm(r.Context(), eventKey, realmID)
		}

		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving reports")
			return
		}

		storeSchema, err := s.Store.GetSchemaByID(r.Context(), *event.SchemaID)
		if _, ok := err.(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving event schema")
			return
		}

		storeMatches, err := s.Store.GetAnalysisInfo(r.Context(), eventKey)
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving match analysis info")
			return
		}

		schema := storeSummaryToSummarySchema(storeSchema)
		teamToMatches := selectTeamMatches(storeMatches, reports)

		teamAnalyses := make([]teamAnalysis, 0)
		for team, teamToMatch := range teamToMatches {
			summary, err := summary.SummarizeTeam(schema, teamToMatch)
			if err != nil {
				ihttp.Error(w, http.StatusInternalServerError)
				s.Logger.WithError(err).WithField("team", team).Error("retrieving match summary")
				return
			}

			teamAnalyses = append(teamAnalyses, teamAnalysisFromSummary(summary, team))
		}

		ihttp.Respond(w, teamAnalyses, http.StatusOK)
	}
}

func (s *Server) matchTeamStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventKey := vars["eventKey"]
		partialMatchKey := vars["matchKey"]
		teamKey := vars["teamKey"]

		var realmID *int64
		userRealmID, realmErr := ihttp.GetRealmID(r)
		if realmErr != nil {
			realmID = &userRealmID
		}

		event, err := s.Store.GetEventForRealm(r.Context(), eventKey, realmID)
		if _, ok := err.(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving event")
			return
		}

		// TODO: handle this by crafting a better query
		// if !s.checkEventAccess(event.RealmID, r) {
		// 	ihttp.Error(w, http.StatusForbidden)
		// 	return
		// }

		if event.SchemaID == nil {
			ihttp.Respond(w, fmt.Errorf("no schema found"), http.StatusBadRequest)
			return
		}

		// Add eventKey as prefix to matchKey so that matchKey is globally
		// unique and consistent with TBA match keys.
		matchKey := fmt.Sprintf("%s_%s", eventKey, partialMatchKey)
		match, err := s.Store.GetMatch(r.Context(), matchKey)
		if _, ok := err.(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving match")
			return
		}

		var reports []store.Report

		if ihttp.GetRoles(r).IsSuperAdmin {
			reports, err = s.Store.GetMatchTeamReports(r.Context(), eventKey, teamKey)
		} else {
			reports, err = s.Store.GetMatchTeamReportsForRealm(r.Context(), eventKey, teamKey, realmID)
		}

		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving reports")
			return
		}
		storeSchema, err := s.Store.GetSchemaByID(r.Context(), *event.SchemaID)
		if _, ok := err.(store.ErrNoResults); ok {
			ihttp.Error(w, http.StatusNotFound)
			return
		} else if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).Error("retrieving event schema")
			return
		}

		schema := storeSummaryToSummarySchema(storeSchema)
		teamToMatches := selectTeamMatches([]store.Match{match}, reports)

		summary, err := summary.SummarizeTeam(schema, teamToMatches[teamKey])
		if err != nil {
			ihttp.Error(w, http.StatusInternalServerError)
			s.Logger.WithError(err).WithField("team", teamKey).Error("retrieving match summary")
			return
		}

		teamAnalysis := teamAnalysisFromSummary(summary, teamKey)
		ihttp.Respond(w, teamAnalysis, http.StatusOK)
	}
}

func selectTeamMatches(storeMatches []store.Match, reports []store.Report) map[string][]summary.Match {
	teamToMatchToReports := make(map[string]map[string][]summary.Report)
	for _, report := range reports {
		var summaryReport summary.Report

		for _, stat := range report.Data {
			summaryReport = append(summaryReport, summary.ReportField{
				Name:  stat.Name,
				Value: stat.Value,
			})
		}

		_, ok := teamToMatchToReports[report.TeamKey]
		if !ok {
			teamToMatchToReports[report.TeamKey] = make(map[string][]summary.Report)
		}

		teamToMatchToReports[report.TeamKey][report.MatchKey] = append(teamToMatchToReports[report.TeamKey][report.MatchKey], summaryReport)
	}

	teamToMatches := make(map[string][]summary.Match)
	for _, storeMatch := range storeMatches {
		teams := append([]string(storeMatch.RedAlliance), []string(storeMatch.BlueAlliance)...)
		for i, team := range teams {
			position := (i % len(storeMatch.RedAlliance)) + 1
			breakdown := storeMatch.RedScoreBreakdown
			if i >= len(storeMatch.RedAlliance) {
				breakdown = storeMatch.BlueScoreBreakdown
			}

			match := summary.Match{
				Key:            storeMatch.Key,
				RobotPosition:  position,
				ScoreBreakdown: summary.ScoreBreakdown(breakdown),
				Reports:        teamToMatchToReports[team][storeMatch.Key],
			}

			teamToMatches[team] = append(teamToMatches[team], match)
		}
	}

	return teamToMatches
}

func storeSummaryToSummarySchema(storeSchema store.Schema) summary.Schema {
	var schema summary.Schema
	for _, statDescription := range storeSchema.Schema {
		field := summary.SchemaField{
			FieldDescriptor: summary.FieldDescriptor{Name: statDescription.FieldDescriptor.Name},
			ReportReference: statDescription.ReportReference,
			TBAReference:    statDescription.TBAReference,
		}

		for _, v := range statDescription.Sum {
			field.Sum = append(field.Sum, summary.FieldDescriptor{Name: v.Name})
		}

		for _, v := range statDescription.AnyOf {
			field.AnyOf = append(field.AnyOf, summary.EqualExpression{
				FieldDescriptor: summary.FieldDescriptor{Name: v.Name},
				Equals:          v.Equals,
			})
		}

		schema = append(schema, field)
	}
	return schema
}

type teamAnalysis struct {
	Team    string        `json:"team"`
	Summary []summaryStat `json:"summary"`
}

type summaryStat struct {
	Name    string  `json:"name"`
	Max     float64 `json:"max"`
	Average float64 `json:"avg"`
}

func teamAnalysisFromSummary(summary summary.Summary, team string) teamAnalysis {
	var stats []summaryStat
	for _, stat := range summary {
		stats = append(stats, summaryStat{
			Name:    stat.Name,
			Max:     stat.Max,
			Average: stat.Average,
		})
	}

	return teamAnalysis{
		Team:    team,
		Summary: stats,
	}
}
