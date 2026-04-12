package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListCountries(c *gin.Context) {
	countries, err := h.references.ListCountries(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, countries)
}

func (h *Handler) ListTimezones(c *gin.Context) {
	country := strings.TrimSpace(c.Query("country"))
	if country == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_country"})
		return
	}
	timezones, err := h.references.ListTimezonesByCountry(c.Request.Context(), strings.ToUpper(country))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, timezones)
}

func (h *Handler) ListCurrencies(c *gin.Context) {
	currencies, err := h.references.ListCurrencies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, currencies)
}
