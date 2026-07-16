package handlers

import (
	"encoding/json"
	"net/http"
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

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	database.DB.Model(&org).Updates(body)
	database.DB.First(&org, org.ID)
	utils.JSON(w, http.StatusOK, org)
}
