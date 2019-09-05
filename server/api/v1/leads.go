package v1

import (
	"net/http"
)

// GetLeads is a queryable endpoint that pulls Leads from the database
func GetLeads(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, `{"message": "not implemented"}`, 500)
}

// CreateLead allows the creation of a lead
func CreateLead(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, `{"message": "not implemented"}`, 500)
}

// UpdateLead allows a Lead to be updated.
func UpdateLead(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, `{"message": "not implemented"}`, 500)
}
