package handlers

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tehilla-22/b2b-api/internal/database"
	"github.com/tehilla-22/b2b-api/internal/middleware"
	"github.com/tehilla-22/b2b-api/internal/models"
	"github.com/tehilla-22/b2b-api/internal/utils"
	"gorm.io/gorm"
)

func uploadFile(file multipart.File, header *multipart.FileHeader, folder string) (string, error) {
	uploadDir := filepath.Join("uploads", folder)
	os.MkdirAll(uploadDir, 0755)

	ext := filepath.Ext(header.Filename)
	filename := uuid.New().String() + ext
	filePath := filepath.Join(uploadDir, filename)

	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	io.Copy(dst, file)
	return "/" + filepath.ToSlash(filePath), nil
}

type ProductHandler struct{}

func NewProductHandler() *ProductHandler {
	return &ProductHandler{}
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	search := r.URL.Query().Get("search")
	seller := r.URL.Query().Get("seller")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	query := database.DB.Model(&models.Product{}).Where("status = ?", "active")
	if category != "" {
		if catID, err := uuid.Parse(category); err == nil {
			query = query.Where("category_id = ?", catID)
		}
	}
	if search != "" {
		query = query.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ?", "%"+strings.ToLower(search)+"%", "%"+strings.ToLower(search)+"%")
	}
	if seller != "" {
		query = query.Where("seller_id = ?", seller)
	}

	var total int64
	query.Count(&total)

	var products []models.Product
	query.Preload("Category").Preload("Images").Preload("Seller", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, username, email")
	}).Offset(offset).Limit(limit).Order("created_at DESC").Find(&products)

	count := int(total)
	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"results": products,
		"count":   count,
		"page":    page,
		"total_pages": (count + limit - 1) / limit,
	})
}

func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var product models.Product
	if err := database.DB.Preload("Category").Preload("Images").Preload("Seller").First(&product, "id = ?", id).Error; err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "Product not found")
		return
	}
	utils.JSON(w, http.StatusOK, product)
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var input struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Currency    string  `json:"currency"`
		MOQ         int     `json:"moq"`
		Stock       int     `json:"stock"`
		CategoryID  string  `json:"category_id"`
		Tags        string  `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if input.CategoryID == "" {
		utils.ErrorJSON(w, http.StatusUnprocessableEntity, "Category is required")
		return
	}
	catID, err := uuid.Parse(input.CategoryID)
	if err != nil {
		utils.ErrorJSON(w, http.StatusUnprocessableEntity, "Invalid category ID")
		return
	}

	slug := strings.ToLower(strings.ReplaceAll(input.Name, " ", "-"))

	product := models.Product{
		SellerID:    uuid.MustParse(sellerID),
		CategoryID:  catID,
		Name:        input.Name,
		Slug:        slug,
		Description: input.Description,
		Price:       input.Price,
		Currency:    input.Currency,
		MOQ:         input.MOQ,
		Stock:       input.Stock,
		InStock:     input.Stock > 0,
		Status:      "active",
		Tags:        input.Tags,
	}

	if err := database.DB.Create(&product).Error; err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to create product")
		return
	}

	database.DB.Preload("Category").First(&product, product.ID)
	utils.JSON(w, http.StatusCreated, product)
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(middleware.UserIDKey).(string)
	id := chi.URLParam(r, "id")

	var product models.Product
	if err := database.DB.First(&product, "id = ? AND seller_id = ?", id, sellerID).Error; err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "Product not found")
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	allowed := map[string]bool{
		"name": true, "description": true, "price": true, "currency": true,
		"moq": true, "stock": true, "category_id": true, "tags": true, "status": true,
	}
	updates := make(map[string]interface{})
	for k, v := range body {
		if allowed[k] {
			updates[k] = v
		}
	}

	database.DB.Model(&product).Updates(updates)
	database.DB.First(&product, product.ID)
	utils.JSON(w, http.StatusOK, product)
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(middleware.UserIDKey).(string)
	id := chi.URLParam(r, "id")

	result := database.DB.Where("id = ? AND seller_id = ?", id, sellerID).Delete(&models.Product{})
	if result.RowsAffected == 0 {
		utils.ErrorJSON(w, http.StatusNotFound, "Product not found")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{"detail": "Product deleted"})
}

func (h *ProductHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(middleware.UserIDKey).(string)
	id := chi.URLParam(r, "id")

	var product models.Product
	if err := database.DB.First(&product, "id = ? AND seller_id = ?", id, sellerID).Error; err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "Product not found")
		return
	}

	r.ParseMultipartForm(10 << 20)
	file, header, err := r.FormFile("image")
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Image file required")
		return
	}
	defer file.Close()

	url, err := uploadFile(file, header, "products")
	if err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to upload image")
		return
	}

	image := models.ProductImage{
		ProductID: product.ID,
		URL:       url,
		IsPrimary: false,
	}
	database.DB.Create(&image)

	database.DB.Preload("Images").First(&product, product.ID)
	utils.JSON(w, http.StatusCreated, product)
}

func (h *ProductHandler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(middleware.UserIDKey).(string)
	productID := chi.URLParam(r, "id")
	imageID := chi.URLParam(r, "imageId")

	var product models.Product
	if err := database.DB.First(&product, "id = ? AND seller_id = ?", productID, sellerID).Error; err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "Product not found")
		return
	}

	result := database.DB.Where("id = ? AND product_id = ?", imageID, productID).Delete(&models.ProductImage{})
	if result.RowsAffected == 0 {
		utils.ErrorJSON(w, http.StatusNotFound, "Image not found")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{"detail": "Image deleted"})
}

func (h *ProductHandler) ListFeatured(w http.ResponseWriter, r *http.Request) {
	var products []models.Product
	database.DB.Where("status = ? AND is_featured = ?", "active", true).
		Preload("Category").Preload("Images").
		Preload("Seller", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, username, email")
		}).
		Order("created_at DESC").Limit(20).Find(&products)

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"results": products,
		"count":   len(products),
	})
}

func (h *ProductHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	var categories []models.Category
	database.DB.Preload("Subcategories").Where("parent_id IS NULL").Order("name ASC").Find(&categories)
	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"results": categories,
		"count":   len(categories),
	})
}

func (h *ProductHandler) ListMyProducts(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var products []models.Product
	database.DB.Preload("Category").Preload("Images").
		Where("seller_id = ?", sellerID).
		Order("created_at DESC").
		Find(&products)

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"results": products,
		"count":   len(products),
	})
}
