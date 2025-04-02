package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Krchnk/EffectiveMobileFullNameTest/internal/db"
	"github.com/Krchnk/EffectiveMobileFullNameTest/internal/models"
	"github.com/sirupsen/logrus"
)

type EnrichmentService struct{}

func (s *EnrichmentService) EnrichPerson(person *models.Person) error {
	logrus.WithField("name", person.Name).Info("Starting enrichment process for person")

	ageAPI := db.GetEnv("AGE_API_URL", "https://api.agify.io")
	if age, err := getAge(person.Name, ageAPI); err == nil {
		person.Age = &age
		logrus.WithField("name", person.Name).Debug("Successfully enriched with age")
	} else {
		logrus.WithFields(logrus.Fields{
			"name":  person.Name,
			"error": err,
		}).Warn("Failed to enrich age")
	}

	genderAPI := db.GetEnv("GENDER_API_URL", "https://api.genderize.io")
	if gender, err := getGender(person.Name, genderAPI); err == nil {
		person.Gender = &gender
		logrus.WithField("name", person.Name).Debug("Successfully enriched with gender")
	} else {
		logrus.WithFields(logrus.Fields{
			"name":  person.Name,
			"error": err,
		}).Warn("Failed to enrich gender")
	}

	nationalityAPI := db.GetEnv("NATIONALITY_API_URL", "https://api.nationalize.io")
	if nationality, err := getNationality(person.Name, nationalityAPI); err == nil {
		person.Nationality = &nationality
		logrus.WithField("name", person.Name).Debug("Successfully enriched with nationality")
	} else {
		logrus.WithFields(logrus.Fields{
			"name":  person.Name,
			"error": err,
		}).Warn("Failed to enrich nationality")
	}

	logrus.WithField("name", person.Name).Info("Completed enrichment process for person")
	return nil
}

func getAge(name, apiURL string) (int, error) {
	logrus.WithField("name", name).Debug("Fetching age")
	resp, err := http.Get(apiURL + "/?name=" + name)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var result struct {
		Age int `json:"age"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if result.Age == 0 {
		return 0, fmt.Errorf("no age prediction available")
	}

	logrus.WithFields(logrus.Fields{
		"name": name,
		"age":  result.Age,
	}).Debug("Age successfully retrieved")
	return result.Age, nil
}

func getGender(name, apiURL string) (string, error) {
	logrus.WithField("name", name).Debug("Fetching gender")
	resp, err := http.Get(apiURL + "/?name=" + name)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var result struct {
		Gender      string  `json:"gender"`
		Probability float64 `json:"probability"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Gender == "" {
		return "", fmt.Errorf("no gender prediction available")
	}

	logrus.WithFields(logrus.Fields{
		"name":   name,
		"gender": result.Gender,
	}).Debug("Gender successfully retrieved")
	return result.Gender, nil
}

func getNationality(name, apiURL string) (string, error) {
	logrus.WithField("name", name).Debug("Fetching nationality")
	resp, err := http.Get(apiURL + "/?name=" + name)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var result struct {
		Country []struct {
			CountryID   string  `json:"country_id"`
			Probability float64 `json:"probability"`
		} `json:"country"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Country) == 0 {
		return "", fmt.Errorf("no nationality prediction available")
	}

	nationality := result.Country[0].CountryID
	logrus.WithFields(logrus.Fields{
		"name":        name,
		"nationality": nationality,
	}).Debug("Nationality successfully retrieved")
	return nationality, nil
}
