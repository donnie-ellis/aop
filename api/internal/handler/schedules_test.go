package handler_test

import (
	"net/http"
	"testing"
)

func TestCreateSchedule_Success(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)

	resp := do(t, http.MethodPost, "/schedules", map[string]any{
		"name":        "nightly-deploy",
		"template_id": tmpl.ID.String(),
		"cron_expr":   "0 2 * * *",
		"timezone":    "America/New_York",
		"enabled":     true,
	}, userJWT)
	assertStatus(t, resp, http.StatusCreated)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["id"] == "" {
		t.Fatal("expected id in response")
	}
	if body["cron_expr"] != "0 2 * * *" {
		t.Fatalf("expected cron_expr, got %v", body["cron_expr"])
	}
	if body["timezone"] != "America/New_York" {
		t.Fatalf("expected timezone, got %v", body["timezone"])
	}
}

func TestCreateSchedule_DefaultTimezone(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)

	resp := do(t, http.MethodPost, "/schedules", map[string]any{
		"name":        "no-tz",
		"template_id": tmpl.ID.String(),
		"cron_expr":   "0 3 * * *",
		// timezone omitted — should default to UTC
	}, userJWT)
	assertStatus(t, resp, http.StatusCreated)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["timezone"] != "UTC" {
		t.Fatalf("expected timezone=UTC, got %v", body["timezone"])
	}
}

func TestCreateSchedule_MissingFields(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)

	cases := []map[string]any{
		{"template_id": tmpl.ID.String(), "cron_expr": "0 * * * *"},  // missing name
		{"name": "x", "cron_expr": "0 * * * *"},                      // missing template_id
		{"name": "x", "template_id": tmpl.ID.String()},               // missing cron_expr
	}
	for _, body := range cases {
		resp := do(t, http.MethodPost, "/schedules", body, userJWT)
		assertStatus(t, resp, http.StatusBadRequest)
		resp.Body.Close()
	}
}

func TestCreateSchedule_InvalidTemplateID(t *testing.T) {
	resp := do(t, http.MethodPost, "/schedules", map[string]any{
		"name":        "bad-tmpl",
		"template_id": "not-a-uuid",
		"cron_expr":   "0 * * * *",
	}, userJWT)
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

func TestCreateSchedule_RequiresAuth(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)
	resp := do(t, http.MethodPost, "/schedules", map[string]any{
		"name": "x", "template_id": tmpl.ID.String(), "cron_expr": "0 * * * *",
	}, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestGetSchedule(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)

	cr := do(t, http.MethodPost, "/schedules", map[string]any{
		"name":        "get-me",
		"template_id": tmpl.ID.String(),
		"cron_expr":   "0 4 * * *",
	}, userJWT)
	assertStatus(t, cr, http.StatusCreated)
	var created map[string]any
	decodeJSON(t, cr, &created)
	id := created["id"].(string)

	resp := do(t, http.MethodGet, "/schedules/"+id, nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["id"] != id {
		t.Fatalf("expected id %s, got %v", id, body["id"])
	}
}

func TestGetSchedule_NotFound(t *testing.T) {
	resp := do(t, http.MethodGet, "/schedules/00000000-0000-0000-0000-000000000030", nil, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestListSchedules(t *testing.T) {
	resp := do(t, http.MethodGet, "/schedules", nil, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var schedules []map[string]any
	decodeJSON(t, resp, &schedules)
}

func TestUpdateSchedule(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)

	cr := do(t, http.MethodPost, "/schedules", map[string]any{
		"name":        "update-me",
		"template_id": tmpl.ID.String(),
		"cron_expr":   "0 5 * * *",
	}, userJWT)
	assertStatus(t, cr, http.StatusCreated)
	var created map[string]any
	decodeJSON(t, cr, &created)
	id := created["id"].(string)

	resp := do(t, http.MethodPut, "/schedules/"+id, map[string]any{
		"name":      "updated-schedule",
		"cron_expr": "0 6 * * *",
		"timezone":  "UTC",
		"enabled":   false,
	}, userJWT)
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["name"] != "updated-schedule" {
		t.Fatalf("expected updated-schedule, got %v", body["name"])
	}
	if body["cron_expr"] != "0 6 * * *" {
		t.Fatalf("expected updated cron_expr, got %v", body["cron_expr"])
	}
}

func TestUpdateSchedule_NotFound(t *testing.T) {
	resp := do(t, http.MethodPut, "/schedules/00000000-0000-0000-0000-000000000031", map[string]any{
		"name": "x", "cron_expr": "0 * * * *", "timezone": "UTC",
	}, userJWT)
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestDeleteSchedule(t *testing.T) {
	p := mustCreateProject(t)
	tmpl := mustCreateTemplate(t, p.ID)

	cr := do(t, http.MethodPost, "/schedules", map[string]any{
		"name":        "delete-me",
		"template_id": tmpl.ID.String(),
		"cron_expr":   "0 7 * * *",
	}, userJWT)
	assertStatus(t, cr, http.StatusCreated)
	var created map[string]any
	decodeJSON(t, cr, &created)
	id := created["id"].(string)

	resp := do(t, http.MethodDelete, "/schedules/"+id, nil, userJWT)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	resp2 := do(t, http.MethodGet, "/schedules/"+id, nil, userJWT)
	assertStatus(t, resp2, http.StatusNotFound)
	resp2.Body.Close()
}
