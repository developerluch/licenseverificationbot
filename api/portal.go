package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"license-bot-go/db"
)

// ── Profile Handlers ─────────────────────────────────────────────────────────

func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	discordID := r.PathValue("discordID")
	if discordID == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	profile, err := s.db.GetOrCreateProfile(ctx, discordID)
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Enrich with comp tier and manager info
	type response struct {
		*db.AgentProfile
		CompTier *db.CompTier `json:"compTier,omitempty"`
		Manager  *struct {
			ID        string `json:"id"`
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		} `json:"manager,omitempty"`
	}
	resp := response{AgentProfile: profile}

	if profile.CompTierID != nil {
		tiers, _ := s.db.ListCompTiers(ctx)
		for _, t := range tiers {
			if t.ID == *profile.CompTierID {
				resp.CompTier = &t
				break
			}
		}
	}

	if profile.ManagerID != nil && *profile.ManagerID != "" {
		// Look up manager name from onboarding_agents
		mProfile, _ := s.db.GetOrCreateProfile(ctx, *profile.ManagerID)
		if mProfile != nil {
			resp.Manager = &struct {
				ID        string `json:"id"`
				FirstName string `json:"firstName"`
				LastName  string `json:"lastName"`
			}{ID: *profile.ManagerID}
			// Get name from onboarding_agents via the agents list
			// We'll do a direct lookup
			if agent, _ := s.db.GetAgentByDiscordIDStr(ctx, *profile.ManagerID); agent != nil {
				resp.Manager.FirstName = agent.FirstName
				resp.Manager.LastName = agent.LastName
			}
		}
	}

	jsonOK(w, resp)
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	discordID := r.PathValue("discordID")
	if discordID == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	profile, err := s.db.UpdateProfile(ctx, discordID, updates)
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, profile)
}

func (s *Server) handleSetManager(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	discordID := r.PathValue("discordID")
	if discordID == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	var body struct {
		ManagerID *string `json:"managerId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	profile, err := s.db.SetManager(ctx, discordID, body.ManagerID)
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, profile)
}

func (s *Server) handleSetCompTier(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	discordID := r.PathValue("discordID")
	if discordID == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	var body struct {
		CompTierID *string `json:"compTierId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	profile, err := s.db.SetCompTier(ctx, discordID, body.CompTierID)
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, profile)
}

// ── Lead Handlers ────────────────────────────────────────────────────────────

func (s *Server) handleListAgentLeads(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	discordID := r.PathValue("discordID")
	if discordID == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	var statusFilter *string
	if st := r.URL.Query().Get("status"); st != "" {
		statusFilter = &st
	}

	leads, err := s.db.ListAgentLeads(ctx, discordID, statusFilter)
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if leads == nil {
		leads = []db.AgentLead{}
	}
	jsonOK(w, leads)
}

func (s *Server) handleCreateAgentLead(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	discordID := r.PathValue("discordID")
	if discordID == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	var body struct {
		FirstName string  `json:"firstName"`
		LastName  string  `json:"lastName"`
		Email     *string `json:"email"`
		Phone     *string `json:"phone"`
		Source    *string `json:"source"`
		Notes     *string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.FirstName == "" || body.LastName == "" {
		jsonError(w, "firstName and lastName are required", http.StatusBadRequest)
		return
	}

	lead, err := s.db.CreateAgentLead(ctx, db.AgentLead{
		AgentID:   discordID,
		FirstName: body.FirstName,
		LastName:  body.LastName,
		Email:     body.Email,
		Phone:     body.Phone,
		Source:    body.Source,
		Notes:     body.Notes,
	})
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(lead)
}

func (s *Server) handleUpdateAgentLead(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	discordID := r.PathValue("discordID")
	leadID := r.PathValue("leadID")
	if discordID == "" || leadID == "" {
		jsonError(w, "missing discordID or leadID", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	lead, err := s.db.UpdateAgentLead(ctx, leadID, discordID, updates)
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, lead)
}

func (s *Server) handleDeleteAgentLead(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	discordID := r.PathValue("discordID")
	leadID := r.PathValue("leadID")
	if discordID == "" || leadID == "" {
		jsonError(w, "missing discordID or leadID", http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteAgentLead(ctx, leadID, discordID); err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Training Handlers ────────────────────────────────────────────────────────

func (s *Server) handleListTrainingItems(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	discordID := r.PathValue("discordID")
	if discordID == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	items, err := s.db.ListTrainingItems(ctx, discordID)
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if items == nil {
		items = []db.AgentTrainingItem{}
	}
	jsonOK(w, items)
}

func (s *Server) handleCreateTrainingItem(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	discordID := r.PathValue("discordID")
	if discordID == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	var body struct {
		Title       string  `json:"title"`
		Description *string `json:"description"`
		DueDate     *string `json:"dueDate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.Title == "" {
		jsonError(w, "title is required", http.StatusBadRequest)
		return
	}

	item, err := s.db.CreateTrainingItem(ctx, db.AgentTrainingItem{
		AgentID:     discordID,
		Title:       body.Title,
		Description: body.Description,
		DueDate:     body.DueDate,
	})
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func (s *Server) handleUpdateTrainingItem(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	discordID := r.PathValue("discordID")
	itemID := r.PathValue("itemID")
	if discordID == "" || itemID == "" {
		jsonError(w, "missing discordID or itemID", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	item, err := s.db.UpdateTrainingItem(ctx, itemID, discordID, updates)
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, item)
}

func (s *Server) handleDeleteTrainingItem(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	discordID := r.PathValue("discordID")
	itemID := r.PathValue("itemID")
	if discordID == "" || itemID == "" {
		jsonError(w, "missing discordID or itemID", http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteTrainingItem(ctx, itemID, discordID); err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Schedule Handlers ────────────────────────────────────────────────────────

func (s *Server) handleListScheduleEvents(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	discordID := r.PathValue("discordID")
	if discordID == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	var startTime, endTime *time.Time
	if st := r.URL.Query().Get("start"); st != "" {
		if t, err := time.Parse(time.RFC3339, st); err == nil {
			startTime = &t
		}
	}
	if et := r.URL.Query().Get("end"); et != "" {
		if t, err := time.Parse(time.RFC3339, et); err == nil {
			endTime = &t
		}
	}

	events, err := s.db.ListScheduleEvents(ctx, discordID, startTime, endTime)
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if events == nil {
		events = []db.AgentScheduleEvent{}
	}
	jsonOK(w, events)
}

func (s *Server) handleCreateScheduleEvent(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	discordID := r.PathValue("discordID")
	if discordID == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	var body struct {
		Title       string  `json:"title"`
		Description *string `json:"description"`
		StartTime   string  `json:"startTime"`
		EndTime     string  `json:"endTime"`
		Type        string  `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.Title == "" || body.StartTime == "" || body.EndTime == "" {
		jsonError(w, "title, startTime, and endTime are required", http.StatusBadRequest)
		return
	}

	startT, err := time.Parse(time.RFC3339, body.StartTime)
	if err != nil {
		jsonError(w, "invalid startTime format (use RFC3339)", http.StatusBadRequest)
		return
	}
	endT, err := time.Parse(time.RFC3339, body.EndTime)
	if err != nil {
		jsonError(w, "invalid endTime format (use RFC3339)", http.StatusBadRequest)
		return
	}

	event, err := s.db.CreateScheduleEvent(ctx, db.AgentScheduleEvent{
		AgentID:     discordID,
		Title:       body.Title,
		Description: body.Description,
		StartTime:   startT,
		EndTime:     endT,
		Type:        body.Type,
	})
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}

func (s *Server) handleUpdateScheduleEvent(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	discordID := r.PathValue("discordID")
	eventID := r.PathValue("eventID")
	if discordID == "" || eventID == "" {
		jsonError(w, "missing discordID or eventID", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	event, err := s.db.UpdateScheduleEvent(ctx, eventID, discordID, updates)
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, event)
}

func (s *Server) handleDeleteScheduleEvent(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	discordID := r.PathValue("discordID")
	eventID := r.PathValue("eventID")
	if discordID == "" || eventID == "" {
		jsonError(w, "missing discordID or eventID", http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteScheduleEvent(ctx, eventID, discordID); err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Org Chart Handlers ───────────────────────────────────────────────────────

func (s *Server) handleGetOrgChart(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := s.db.GetOrgChart(ctx)
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform to OrgChartNode shape expected by the frontend
	type compTierJSON struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Percentage int    `json:"percentage"`
		Order      int    `json:"order"`
	}
	type profileJSON struct {
		ManagerID *string       `json:"managerId"`
		CompTier  *compTierJSON `json:"compTier,omitempty"`
	}
	type stageJSON struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	type nodeJSON struct {
		ID        string       `json:"id"`
		FirstName string       `json:"firstName"`
		LastName  string       `json:"lastName"`
		Profile   *profileJSON `json:"profile,omitempty"`
		Stage     *stageJSON   `json:"stage,omitempty"`
	}

	nodes := make([]nodeJSON, 0, len(rows))
	for _, row := range rows {
		node := nodeJSON{
			ID:        row.AgentID,
			FirstName: row.FirstName,
			LastName:  row.LastName,
		}

		// Always include profile if we have any profile data
		p := &profileJSON{ManagerID: row.ManagerID}
		if row.TierName != nil && row.CompTierID != nil {
			pct := 0
			ord := 0
			if row.TierPct != nil {
				pct = *row.TierPct
			}
			if row.TierOrder != nil {
				ord = *row.TierOrder
			}
			p.CompTier = &compTierJSON{
				ID:         *row.CompTierID,
				Name:       *row.TierName,
				Percentage: pct,
				Order:      ord,
			}
		}
		node.Profile = p

		if row.StageName != nil && row.StageColor != nil {
			node.Stage = &stageJSON{Name: *row.StageName, Color: *row.StageColor}
		}

		nodes = append(nodes, node)
	}

	jsonOK(w, nodes)
}

func (s *Server) handleAssignManager(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var body struct {
		AgentID   string `json:"agentId"`
		ManagerID string `json:"managerId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.AgentID == "" {
		jsonError(w, "agentId is required", http.StatusBadRequest)
		return
	}

	var managerPtr *string
	if body.ManagerID != "" {
		managerPtr = &body.ManagerID
	}

	_, err := s.db.SetManager(ctx, body.AgentID, managerPtr)
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "ok"})
}

// ── Comp Tier Handlers ───────────────────────────────────────────────────────

func (s *Server) handleListCompTiers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tiers, err := s.db.ListCompTiers(ctx)
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if tiers == nil {
		tiers = []db.CompTier{}
	}

	// Enrich each tier with its agent profiles
	type tierWithProfiles struct {
		db.CompTier
		Profiles []struct {
			AgentID string `json:"agentId"`
			Agent   struct {
				FirstName string `json:"firstName"`
				LastName  string `json:"lastName"`
			} `json:"agent"`
		} `json:"profiles"`
	}

	result := make([]tierWithProfiles, 0, len(tiers))
	for _, t := range tiers {
		tp := tierWithProfiles{CompTier: t}
		profiles, _ := s.db.GetCompTierProfiles(ctx, t.ID)
		tp.Profiles = make([]struct {
			AgentID string `json:"agentId"`
			Agent   struct {
				FirstName string `json:"firstName"`
				LastName  string `json:"lastName"`
			} `json:"agent"`
		}, 0, len(profiles))
		for _, p := range profiles {
			tp.Profiles = append(tp.Profiles, struct {
				AgentID string `json:"agentId"`
				Agent   struct {
					FirstName string `json:"firstName"`
					LastName  string `json:"lastName"`
				} `json:"agent"`
			}{
				AgentID: p.AgentID,
				Agent: struct {
					FirstName string `json:"firstName"`
					LastName  string `json:"lastName"`
				}{FirstName: p.FirstName, LastName: p.LastName},
			})
		}
		result = append(result, tp)
	}

	jsonOK(w, result)
}

func (s *Server) handleCreateCompTier(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var body struct {
		Name       string `json:"name"`
		Percentage int    `json:"percentage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}

	tier, err := s.db.CreateCompTier(ctx, body.Name, body.Percentage)
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tier)
}

func (s *Server) handleUpdateCompTier(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tierID := r.PathValue("tierID")
	if tierID == "" {
		jsonError(w, "missing tierID", http.StatusBadRequest)
		return
	}

	var body struct {
		Name       string `json:"name"`
		Percentage int    `json:"percentage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tier, err := s.db.UpdateCompTier(ctx, tierID, body.Name, body.Percentage)
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, tier)
}

func (s *Server) handleDeleteCompTier(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tierID := r.PathValue("tierID")
	if tierID == "" {
		jsonError(w, "missing tierID", http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteCompTier(ctx, tierID); err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleReorderCompTiers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var body struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(body.IDs) == 0 {
		jsonError(w, "ids array is required", http.StatusBadRequest)
		return
	}

	if err := s.db.ReorderCompTiers(ctx, body.IDs); err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "ok"})
}
