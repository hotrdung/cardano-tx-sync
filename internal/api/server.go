package api

import (
	"cardano-tx-sync/internal/chainsync"
	"cardano-tx-sync/internal/model"
	"cardano-tx-sync/internal/storage"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server holds the dependencies for the API server.
type Server struct {
	storage storage.Storage
	syncer  *chainsync.Syncer
	logger  *zap.Logger
	router  *gin.Engine
}

var validMappingTypes = map[model.MappingType]bool{
	model.MappingTypeAddress:  true,
	model.MappingTypePolicyID: true,
	model.MappingTypeAnyCert:  true,
	model.MappingTypeCertType: true,
	model.MappingTypeProposal: true,
	model.MappingTypeVote:     true,
}

// NewServer creates a new API server.
func NewServer(storage storage.Storage, syncer *chainsync.Syncer, logger *zap.Logger) *Server {
	server := &Server{
		storage: storage,
		syncer:  syncer,
		logger:  logger,
	}
	server.setupRouter()
	return server
}

func (s *Server) setupRouter() {
	router := gin.Default()

	mappings := router.Group("/mappings")
	{
		mappings.POST("", s.addMapping)
		mappings.DELETE("/:id", s.removeMapping)
	}

	sync := router.Group("/sync")
	{
		sync.POST("/start", s.startSync)
	}

	s.router = router
}

// Start runs the HTTP server on a specific address.
func (s *Server) Start(address string) error {
	return s.router.Run(address)
}

func (s *Server) addMapping(c *gin.Context) {
	var req model.Mapping
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !validMappingTypes[req.Type] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mapping type"})
		return
	}

	// For certain types, the key can be a generic value like "any"
	if req.Type == model.MappingTypeAnyCert || req.Type == model.MappingTypeProposal || req.Type == model.MappingTypeVote {
		if req.Key == "" {
			req.Key = "any" // Use a default key
		}
	}

	id, err := s.storage.AddMapping(req)
	if err != nil {
		s.logger.Error("failed to add mapping", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add mapping"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (s *Server) removeMapping(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	err = s.storage.RemoveMapping(id)
	if err != nil {
		s.logger.Error("failed to remove mapping", zap.Error(err), zap.Int("id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove mapping"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) startSync(c *gin.Context) {
	var req struct {
		Slot uint64 `json:"slot"`
		Hash string `json:"hash"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	point := model.Checkpoint{
		Slot: req.Slot,
		Hash: req.Hash,
	}

	if err := s.syncer.SetStartPoint(point); err != nil {
		s.logger.Error("failed to set start point", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set start point"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "sync point updated"})
}
