package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/tehilla-22/b2b-api/internal/config"
	"github.com/tehilla-22/b2b-api/internal/database"
	"github.com/tehilla-22/b2b-api/internal/middleware"
	"github.com/tehilla-22/b2b-api/internal/models"
	"github.com/tehilla-22/b2b-api/internal/utils"
	"github.com/google/uuid"
)

type OrganizationHandler struct {
	cfg *config.Config
}

func NewOrganizationHandler(cfg *config.Config) *OrganizationHandler {
	return &OrganizationHandler{cfg: cfg}
}

func (h *OrganizationHandler) GetMyOrganization(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var org models.Organization
	err := database.DB.
		Joins("JOIN organization_members ON organization_members.organization_id = organizations.id").
		Where("organization_members.user_id = ?", userID).
		First(&org).Error

	if err != nil {
		newOrg := models.Organization{
			Name: "My Company",
		}
		database.DB.Create(&newOrg)
		database.DB.Create(&models.OrganizationMember{
			UserID:         uuid.MustParse(userID),
			OrganizationID: newOrg.ID,
			Role:           "admin",
		})
		utils.JSON(w, http.StatusOK, newOrg)
		return
	}

	utils.JSON(w, http.StatusOK, org)
}

func (h *OrganizationHandler) UpdateMyOrganization(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var org models.Organization
	err := database.DB.
		Joins("JOIN organization_members ON organization_members.organization_id = organizations.id").
		Where("organization_members.user_id = ?", userID).
		First(&org).Error

	if err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "No organization found")
		return
	}

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			utils.ErrorJSON(w, http.StatusBadRequest, "Failed to parse form")
			return
		}

		if name := r.FormValue("name"); name != "" {
			org.Name = name
		}
		if desc := r.FormValue("description"); desc != "" {
			org.Description = desc
		}
		if email := r.FormValue("email"); email != "" {
			org.Email = email
		}
		if phone := r.FormValue("phone_number"); phone != "" {
			org.PhoneNumber = phone
		}
		if website := r.FormValue("website"); website != "" {
			org.Website = website
		}
		if industry := r.FormValue("industry"); industry != "" {
			org.Industry = industry
		}
		if bt := r.FormValue("business_type"); bt != "" {
			org.BusinessType = bt
		}
		if reg := r.FormValue("registration_number"); reg != "" {
			org.RegistrationNumber = reg
		}
		if tax := r.FormValue("tax_id_number"); tax != "" {
			org.TaxIDNumber = tax
		}
		if country := r.FormValue("country"); country != "" {
			org.Country = country
		}
		if state := r.FormValue("state"); state != "" {
			org.State = state
		}
		if city := r.FormValue("city"); city != "" {
			org.City = city
		}
		if addr := r.FormValue("address"); addr != "" {
			org.Address = addr
		}
		if postal := r.FormValue("postal_code"); postal != "" {
			org.PostalCode = postal
		}
		if emp := r.FormValue("number_of_employees"); emp != "" {
			org.NumberOfEmployees = emp
		}
		if year := r.FormValue("year_established"); year != "" {
			if y, err := strconv.Atoi(year); err == nil {
				val := y
				org.YearEstablished = &val
			}
		}

		file, header, err := r.FormFile("logo")
		if err == nil {
			defer file.Close()
			logoURL, err := uploadFile(file, header, "logos")
			if err == nil {
				org.Logo = logoURL
			}
		}

		database.DB.Save(&org)
		utils.JSON(w, http.StatusOK, org)
		return
	}

	var body struct {
		Name               string `json:"name"`
		Description        string `json:"description"`
		Email              string `json:"email"`
		PhoneNumber        string `json:"phone_number"`
		Website            string `json:"website"`
		Industry           string `json:"industry"`
		BusinessType       string `json:"business_type"`
		RegistrationNumber string `json:"registration_number"`
		TaxIDNumber        string `json:"tax_id_number"`
		Country            string `json:"country"`
		State              string `json:"state"`
		City               string `json:"city"`
		Address            string `json:"address"`
		PostalCode         string `json:"postal_code"`
		NumberOfEmployees  string `json:"number_of_employees"`
		YearEstablished    string `json:"year_established"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if v := strings.TrimSpace(body.Name); v != "" { updates["name"] = v }
	if v := strings.TrimSpace(body.Description); v != "" { updates["description"] = v }
	if v := strings.TrimSpace(body.Email); v != "" { updates["email"] = v }
	if v := strings.TrimSpace(body.PhoneNumber); v != "" { updates["phone_number"] = v }
	if v := strings.TrimSpace(body.Website); v != "" { updates["website"] = v }
	if v := strings.TrimSpace(body.Industry); v != "" { updates["industry"] = v }
	if v := strings.TrimSpace(body.BusinessType); v != "" { updates["business_type"] = v }
	if v := strings.TrimSpace(body.RegistrationNumber); v != "" { updates["registration_number"] = v }
	if v := strings.TrimSpace(body.TaxIDNumber); v != "" { updates["tax_id_number"] = v }
	if v := strings.TrimSpace(body.Country); v != "" { updates["country"] = v }
	if v := strings.TrimSpace(body.State); v != "" { updates["state"] = v }
	if v := strings.TrimSpace(body.City); v != "" { updates["city"] = v }
	if v := strings.TrimSpace(body.Address); v != "" { updates["address"] = v }
	if v := strings.TrimSpace(body.PostalCode); v != "" { updates["postal_code"] = v }
	if v := strings.TrimSpace(body.NumberOfEmployees); v != "" { updates["number_of_employees"] = v }
	if v := strings.TrimSpace(body.YearEstablished); v != "" {
		if y, err := strconv.Atoi(v); err == nil {
			updates["year_established"] = y
		}
	}

	if len(updates) == 0 {
		utils.ErrorJSON(w, http.StatusBadRequest, "No fields to update")
		return
	}

	database.DB.Model(&org).Updates(updates)
	database.DB.First(&org, org.ID)
	utils.JSON(w, http.StatusOK, org)
}
