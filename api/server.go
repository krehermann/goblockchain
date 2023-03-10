package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/krehermann/goblockchain/core"
	"github.com/krehermann/goblockchain/types"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type ServerConfig struct {
	ListenerAddr string
	Logger       *zap.Logger
}

type Server struct {
	ServerConfig
	chain  *core.Blockchain
	txChan chan<- *core.Transaction

	logger *zap.Logger
}

func NewServer(config ServerConfig, chain *core.Blockchain, txChan chan *core.Transaction) (*Server, error) {
	if config.Logger == nil {
		config.Logger, _ = zap.NewDevelopment()
	}
	s := &Server{
		ServerConfig: config,
		chain:        chain,
		logger:       config.Logger,
		txChan:       txChan,
	}

	return s, nil
}

func (s *Server) Start() error {
	s.logger.Info("api server starting",
		zap.String("addr", s.ListenerAddr))
	echoer := echo.New()

	echoer.GET("/block/hash/:hash", s.handleGetBlockByHash)
	echoer.GET("/block/height/:height", s.handleGetBlockByHeight)
	echoer.GET("/txn/:hash", s.handleGetTransaction)

	echoer.POST("/txn", s.handlePutTransaction)
	return echoer.Start(s.ListenerAddr)
}

func (s *Server) handleGetBlockByHash(ectx echo.Context) error {
	q := ectx.Param("hash")

	h, err := types.HashFromHex(q)
	if err != nil {
		return ectx.JSON(http.StatusBadRequest,
			map[string]any{
				"error": err.Error(),
			})
	}

	b, err := s.chain.GetBlockHash(h)
	if err != nil {
		return ectx.JSON(http.StatusNotFound,
			map[string]any{
				"error": err.Error(),
			})
	}

	return ectx.JSON(http.StatusOK,
		map[string]any{"msg": "height it works!",
			"query":  ectx.QueryString(),
			"params": ectx.ParamNames(),
			"values": ectx.ParamValues(),
			"val":    q,
			"height": h,
			"block":  b,
		})
}

func (s *Server) handleGetTransaction(ectx echo.Context) error {
	q := ectx.Param("hash")

	h, err := types.HashFromHex(q)
	if err != nil {
		return ectx.JSON(http.StatusBadRequest,
			map[string]any{
				"error": err.Error(),
			})
	}

	txn, err := s.chain.GetTransaction(h)
	if err != nil {
		return ectx.JSON(http.StatusNotFound,
			map[string]any{
				"error": err.Error(),
			})
	}

	return ectx.JSON(http.StatusOK,
		map[string]any{"msg": "transaction",
			"query":  ectx.QueryString(),
			"params": ectx.ParamNames(),
			"values": ectx.ParamValues(),
			"val":    q,
			"txn":    txn,
		})
}

func (s *Server) handlePutTransaction(ectx echo.Context) error {

	b, err := ioutil.ReadAll(ectx.Request().Body)

	if err != nil {
		return ectx.JSON(http.StatusBadRequest,
			map[string]any{
				"b":     b,
				"error": err.Error(),
			})
	}

	txn := new(core.Transaction)
	err = json.Unmarshal(b, txn)

	if err != nil {
		return ectx.JSON(http.StatusBadRequest,
			map[string]any{
				"b":     string(b),
				"error": err.Error(),
			})
	}

	s.txChan <- txn

	return ectx.JSON(http.StatusOK,
		map[string]any{"msg": "transaction",
			"txn": txn,
		})
}

func (s *Server) handleGetBlockByHeight(ectx echo.Context) error {
	val := ectx.Param("height")

	height, err := strconv.Atoi(val)
	if err != nil {
		return ectx.JSON(http.StatusBadRequest,
			map[string]any{
				"error": err.Error(),
			})

	}
	b, err := s.chain.GetBlockAt(uint32(height))
	if err != nil {
		return ectx.JSON(http.StatusNotFound,
			map[string]any{
				"error": err.Error(),
			})
	}

	err = ectx.JSON(http.StatusOK,
		map[string]any{"msg": "height it works!",
			"query":  ectx.QueryString(),
			"params": ectx.ParamNames(),
			"values": ectx.ParamValues(),
			"val":    val,
			"height": height,
			"block":  b,
		})
	if err != nil {
		panic(err)
	}
	return nil
}
