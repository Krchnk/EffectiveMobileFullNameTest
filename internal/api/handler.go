package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/Krchnk/EffectiveMobileFullNameTest/internal/models"
	"github.com/Krchnk/EffectiveMobileFullNameTest/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	_ "github.com/Krchnk/EffectiveMobileFullNameTest/docs"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
)

type Handler struct {
	db     *sql.DB
	enrich *service.EnrichmentService
}

func StartServer(db *sql.DB, addr string) error {
	r := gin.Default()
	h := &Handler{db: db, enrich: &service.EnrichmentService{}}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/persons", h.GetPersons)
	r.POST("/persons", h.CreatePerson)
	r.PATCH("/persons/:id", h.PatchPerson)
	r.PUT("/persons/:id", h.UpdatePerson)
	r.DELETE("/persons/:id", h.DeletePerson)

	logrus.WithField("addr", addr).Info("Server starting")
	return r.Run(addr)
}

// GetPersons godoc
// @Summary Get list of persons
// @Description Returns a paginated list of persons with optional filters
// @Tags persons
// @Accept json
// @Produce json
// @Param limit query int false "Number of items to return" default(10)
// @Param offset query int false "Number of items to skip" default(0)
// @Param name query string false "Filter by name"
// @Param surname query string false "Filter by surname"
// @Param patronymic query string false "Filter by patronymic"
// @Param age query int false "Filter by age"
// @Param gender query string false "Filter by gender" Enums(male, female, other)
// @Param nationality query string false "Filter by nationality"
// @Success 200 {array} models.Person
// @Failure 400 {object} models.ErrorResponse "Invalid parameters"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /persons [get]
func (h *Handler) GetPersons(c *gin.Context) {
	logrus.Info("Received GET /persons request")
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")
	nameFilter := c.Query("name")
	surnameFilter := c.Query("surname")
	patronymicFilter := c.Query("patronymic")
	ageStr := c.Query("age")
	genderFilter := c.Query("gender")
	nationalityFilter := c.Query("nationality")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		logrus.WithField("limit", limitStr).Error("Invalid limit parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		logrus.WithField("offset", offsetStr).Error("Invalid offset parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset parameter"})
		return
	}

	var ageFilter *int
	if ageStr != "" {
		age, err := strconv.Atoi(ageStr)
		if err != nil {
			logrus.WithField("age", ageStr).Error("Invalid age parameter")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid age parameter"})
			return
		}
		ageFilter = &age
	}

	query := "SELECT id, name, surname, patronymic, age, gender, nationality FROM persons WHERE 1=1"
	var args []interface{}
	argCount := 1

	if nameFilter != "" {
		query += " AND name = $" + strconv.Itoa(argCount)
		args = append(args, nameFilter)
		argCount++
	}
	if surnameFilter != "" {
		query += " AND surname = $" + strconv.Itoa(argCount)
		args = append(args, surnameFilter)
		argCount++
	}
	if patronymicFilter != "" {
		query += " AND patronymic = $" + strconv.Itoa(argCount)
		args = append(args, patronymicFilter)
		argCount++
	}
	if ageFilter != nil {
		query += " AND age = $" + strconv.Itoa(argCount)
		args = append(args, *ageFilter)
		argCount++
	}
	if genderFilter != "" {
		query += " AND gender = $" + strconv.Itoa(argCount)
		args = append(args, genderFilter)
		argCount++
	}
	if nationalityFilter != "" {
		query += " AND nationality = $" + strconv.Itoa(argCount)
		args = append(args, nationalityFilter)
		argCount++
	}

	query += " ORDER BY id LIMIT $" + strconv.Itoa(argCount) + " OFFSET $" + strconv.Itoa(argCount+1)
	args = append(args, limit, offset)

	logrus.WithFields(logrus.Fields{
		"query": query,
		"args":  args,
	}).Debug("Executing database query for persons")

	rows, err := h.db.Query(query, args...)
	if err != nil {
		logrus.WithError(err).Error("Database query failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	defer rows.Close()

	var persons []models.Person
	for rows.Next() {
		var p models.Person
		err := rows.Scan(&p.ID, &p.Name, &p.Surname, &p.Patronymic, &p.Age, &p.Gender, &p.Nationality)
		if err != nil {
			logrus.WithError(err).Error("Failed to scan row")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
		persons = append(persons, p)
	}

	logrus.WithField("count", len(persons)).Info("Successfully retrieved persons")
	c.JSON(http.StatusOK, persons)
}

// CreatePerson godoc
// @Summary Create a new person
// @Description Creates a new person and enriches their data with age, gender, and nationality
// @Tags persons
// @Accept json
// @Produce json
// @Param person body models.PersonRequest true "Person data to create"
// @Success 201 {object} models.Person
// @Failure 400 {object} models.ErrorResponse "Invalid request body"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /persons [post]
func (h *Handler) CreatePerson(c *gin.Context) {
	logrus.Info("Received POST /persons request")
	var req models.PersonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithError(err).Error("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	logrus.WithField("request", req).Debug("Parsed person request")

	person := models.Person{
		Name:       req.Name,
		Surname:    req.Surname,
		Patronymic: req.Patronymic,
	}

	logrus.WithField("name", person.Name).Debug("Starting person enrichment")
	if err := h.enrich.EnrichPerson(&person); err != nil {
		logrus.WithError(err).Warn("Failed to enrich person data")
	}

	query := `
		INSERT INTO persons (name, surname, patronymic, age, gender, nationality)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	logrus.WithField("person", person).Debug("Inserting person into database")
	err := h.db.QueryRow(query, person.Name, person.Surname, person.Patronymic,
		person.Age, person.Gender, person.Nationality).Scan(&person.ID)
	if err != nil {
		logrus.WithError(err).Error("Failed to insert person into database")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create person"})
		return
	}

	logrus.WithField("id", person.ID).Info("Person successfully created")
	c.JSON(http.StatusCreated, person)
}

// PatchPerson godoc
// @Summary Partially update a person
// @Description Updates specific fields of an existing person by ID
// @Tags persons
// @Accept json
// @Produce json
// @Param id path int true "Person ID"
// @Param person body models.PersonPatch true "Fields to update"
// @Success 200 {object} models.Person
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 404 {object} models.ErrorResponse "Person not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /persons/{id} [patch]
func (h *Handler) PatchPerson(c *gin.Context) {
	logrus.Info("Received PATCH /persons/:id request")
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.WithField("id", idStr).Error("Invalid ID parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var patch models.PersonPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		logrus.WithError(err).Error("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	logrus.WithFields(logrus.Fields{
		"id":    id,
		"patch": patch,
	}).Debug("Parsed patch person request")

	query := "UPDATE persons SET "
	var args []interface{}
	argCount := 1

	if patch.Name != nil {
		query += "name = $" + strconv.Itoa(argCount) + ", "
		args = append(args, *patch.Name)
		argCount++
	}
	if patch.Surname != nil {
		query += "surname = $" + strconv.Itoa(argCount) + ", "
		args = append(args, *patch.Surname)
		argCount++
	}
	if patch.Patronymic != nil {
		query += "patronymic = $" + strconv.Itoa(argCount) + ", "
		args = append(args, *patch.Patronymic)
		argCount++
	}
	if patch.Age != nil {
		query += "age = $" + strconv.Itoa(argCount) + ", "
		args = append(args, *patch.Age)
		argCount++
	}
	if patch.Gender != nil {
		query += "gender = $" + strconv.Itoa(argCount) + ", "
		args = append(args, *patch.Gender)
		argCount++
	}
	if patch.Nationality != nil {
		query += "nationality = $" + strconv.Itoa(argCount) + ", "
		args = append(args, *patch.Nationality)
		argCount++
	}

	if argCount == 1 {
		logrus.WithField("id", id).Error("No fields to update")
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	query = query[:len(query)-2]
	query += " WHERE id = $" + strconv.Itoa(argCount)
	args = append(args, id)

	logrus.WithField("id", id).Debug("Updating person in database")
	result, err := h.db.Exec(query, args...)
	if err != nil {
		logrus.WithError(err).Error("Failed to update person")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update person"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		logrus.WithField("id", id).Warn("Person not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Person not found"})
		return
	}

	var updatedPerson models.Person
	err = h.db.QueryRow("SELECT id, name, surname, patronymic, age, gender, nationality FROM persons WHERE id = $1", id).
		Scan(&updatedPerson.ID, &updatedPerson.Name, &updatedPerson.Surname, &updatedPerson.Patronymic, &updatedPerson.Age, &updatedPerson.Gender, &updatedPerson.Nationality)
	if err != nil {
		logrus.WithError(err).Error("Failed to fetch updated person")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated person"})
		return
	}

	logrus.WithField("id", id).Info("Person successfully updated")
	c.JSON(http.StatusOK, updatedPerson)
}

// UpdatePerson godoc
// @Summary Update a person
// @Description Updates an existing person by ID
// @Tags persons
// @Accept json
// @Produce json
// @Param id path int true "Person ID"
// @Param person body models.Person true "Updated person data"
// @Success 200 {object} models.Person
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 404 {object} models.ErrorResponse "Person not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /persons/{id} [put]
func (h *Handler) UpdatePerson(c *gin.Context) {
	logrus.Info("Received PUT /persons/:id request")
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.WithField("id", idStr).Error("Invalid ID parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var person models.Person
	if err := c.ShouldBindJSON(&person); err != nil {
		logrus.WithError(err).Error("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	person.ID = id

	logrus.WithFields(logrus.Fields{
		"id":     id,
		"person": person,
	}).Debug("Parsed update person request")

	query := `
		UPDATE persons 
		SET name = $1, surname = $2, patronymic = $3, age = $4, gender = $5, nationality = $6
		WHERE id = $7`

	logrus.WithField("id", id).Debug("Updating person in database")
	result, err := h.db.Exec(query, person.Name, person.Surname, person.Patronymic,
		person.Age, person.Gender, person.Nationality, person.ID)
	if err != nil {
		logrus.WithError(err).Error("Failed to update person")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update person"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		logrus.WithField("id", id).Warn("Person not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Person not found"})
		return
	}

	logrus.WithField("id", id).Info("Person successfully updated")
	c.JSON(http.StatusOK, person)
}

// DeletePerson godoc
// @Summary Delete a person
// @Description Deletes a person by ID
// @Tags persons
// @Param id path int true "Person ID"
// @Success 204 "Person deleted"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Person not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /persons/{id} [delete]
func (h *Handler) DeletePerson(c *gin.Context) {
	logrus.Info("Received DELETE /persons/:id request")
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.WithField("id", idStr).Error("Invalid ID parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	logrus.WithField("id", id).Debug("Preparing to delete person")
	query := "DELETE FROM persons WHERE id = $1"
	result, err := h.db.Exec(query, id)
	if err != nil {
		logrus.WithError(err).Error("Failed to delete person")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete person"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		logrus.WithField("id", id).Warn("Person not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Person not found"})
		return
	}

	logrus.WithField("id", id).Info("Person successfully deleted")
	c.Status(http.StatusNoContent)
}
