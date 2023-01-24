package api

import (
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
	chain *core.Blockchain

	logger *zap.Logger
}

func NewServer(config ServerConfig, chain *core.Blockchain) (*Server, error) {
	if config.Logger == nil {
		config.Logger, _ = zap.NewDevelopment()
	}
	s := &Server{
		ServerConfig: config,
		chain:        chain,
		logger:       config.Logger,
	}

	return s, nil
}

func (s *Server) Start() error {
	s.logger.Info("api server starting",
		zap.String("addr", s.ListenerAddr))
	echoer := echo.New()

	echoer.GET("/block/hash/:hash", s.handleGetBlockByHash)
	echoer.GET("/block/height/:height", s.handleGetBlockByHeight)

	return echoer.Start(s.ListenerAddr)
}

func (s *Server) handleGetBlockByHash(ectx echo.Context) error {
	q := ectx.Param("hash")

	h, err := types.HashFromHex(q)
	if err != nil {
		return err
	}

	b, err := s.chain.GetBlockHash(h)
	if err != nil {
		return err
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

func (s *Server) handleGetBlockByHeight(ectx echo.Context) error {
	val := ectx.Param("height")

	height, err := strconv.Atoi(val)
	if err != nil {
		//panic(err)
		//return err
		return ectx.JSON(http.StatusOK,
			map[string]any{"msg": "height it works!",
				"query":    ectx.QueryString(),
				"params":   ectx.ParamNames(),
				"values":   ectx.ParamValues(),
				"val":      val,
				"height":   height,
				"conv err": err,
			})

	}
	b, err := s.chain.GetBlockAt(uint32(height))
	if err != nil {
		//panic(err)
		//return err
		return ectx.JSON(http.StatusOK,
			map[string]any{"msg": "height it works!",
				"query":  ectx.QueryString(),
				"params": ectx.ParamNames(),
				"values": ectx.ParamValues(),
				"val":    val,
				"height": height,
				//	"block":   b,
				"get err": err,
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
