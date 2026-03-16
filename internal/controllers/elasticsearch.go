package controllers

import (
	"awesomeProject/internal/elasticsearch"
	"awesomeProject/pkg"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// SearchLogs 搜索日志（支持多条件过滤 + 分页）
func SearchLogs(c *gin.Context) {
	var req struct {
		Level     string  `form:"level"`      // 日志级别：INFO, ERROR, WARN
		Service   string  `form:"service"`    // 服务名称
		Category  string  `form:"category"`   // 类别
		DeviceID  string  `form:"device_id"`  // 设备ID
		StartDate string  `form:"start_date"` // 开始日期 YYYY-MM-DD
		EndDate   string  `form:"end_date"`   // 结束日期 YYYY-MM-DD
		Page      int     `form:"page"`
		PageSize  int     `form:"page_size"`
		MinTemp   float64 `form:"min_temp"`
		MaxTemp   float64 `form:"max_temp"`
		MinHum    float64 `form:"min_hum"`
		MaxHum    float64 `form:"max_hum"`
		MinDist   float64 `form:"min_dist"`
		MaxDist   float64 `form:"max_dist"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, "参数错误")
		return
	}

	// 构建 ES 查询
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{},
			},
		},
		"sort": []interface{}{
			map[string]interface{}{
				"@timestamp": map[string]interface{}{
					"order": "desc",
				},
			},
		},
	}

	must := query["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"].([]interface{})

	// 日志级别
	if req.Level != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"level": req.Level,
			},
		})
	}

	// 服务名称
	if req.Service != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"service": req.Service,
			},
		})
	}

	// 设备ID
	if req.DeviceID != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"device_id": req.DeviceID,
			},
		})
	}

	// 类别
	if req.Category != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"category": req.Category,
			},
		})
	}

	// 时间范围
	if req.StartDate != "" || req.EndDate != "" {
		rangeClause := map[string]interface{}{}
		if req.StartDate != "" {
			if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
				rangeClause["gte"] = t.Format(time.RFC3339)
			}
		}
		if req.EndDate != "" {
			if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
				// 结束时间通常取当天 23:59:59
				rangeClause["lte"] = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second).Format(time.RFC3339)
			}
		}
		if len(rangeClause) > 0 {
			must = append(must, map[string]interface{}{
				"range": map[string]interface{}{
					"@timestamp": rangeClause,
				},
			})
		}
	}

	// 温度范围
	if req.MinTemp != 0 || req.MaxTemp != 0 {
		rangeClause := map[string]interface{}{}
		if req.MinTemp != 0 {
			rangeClause["gte"] = req.MinTemp
		}
		if req.MaxTemp != 0 {
			rangeClause["lte"] = req.MaxTemp
		}
		if len(rangeClause) > 0 {
			must = append(must, map[string]interface{}{
				"range": map[string]interface{}{
					"temperature": rangeClause,
				},
			})
		}
	}

	// 湿度范围
	if req.MinHum != 0 || req.MaxHum != 0 {
		rangeClause := map[string]interface{}{}
		if req.MinHum != 0 {
			rangeClause["gte"] = req.MinHum
		}
		if req.MaxHum != 0 {
			rangeClause["lte"] = req.MaxHum
		}
		if len(rangeClause) > 0 {
			must = append(must, map[string]interface{}{
				"range": map[string]interface{}{
					"humidity": rangeClause,
				},
			})
		}
	}

	// 距离范围
	if req.MinDist != 0 || req.MaxDist != 0 {
		rangeClause := map[string]interface{}{}
		if req.MinDist != 0 {
			rangeClause["gte"] = req.MinDist
		}
		if req.MaxDist != 0 {
			rangeClause["lte"] = req.MaxDist
		}
		if len(rangeClause) > 0 {
			must = append(must, map[string]interface{}{
				"range": map[string]interface{}{
					"distance": rangeClause,
				},
			})
		}
	}

	// 分页
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	query["from"] = (req.Page - 1) * req.PageSize
	query["size"] = req.PageSize

	// 执行搜索
	result, err := elasticsearch.ESClient.Search(query)
	if err != nil {
		log.Printf("[SearchLogs] 搜索失败: %v", err)
		pkg.ErrorResponse(c, http.StatusInternalServerError, "搜索失败")
		return
	}

	pkg.RightResponse(c, result, "搜索成功")
}

// GetLogStats 获取统计信息（平均值、最大值、最小值）
func GetLogStats(c *gin.Context) {
	var req struct {
		DeviceID  string `form:"device_id"`
		StartDate string `form:"start_date"`
		EndDate   string `form:"end_date"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, "参数错误")
		return
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{},
			},
		},
		"aggs": map[string]interface{}{
			"avg_temp": map[string]interface{}{
				"avg": map[string]interface{}{
					"field": "temperature",
				},
			},
			"avg_humidity": map[string]interface{}{
				"avg": map[string]interface{}{
					"field": "humidity",
				},
			},
			"avg_distance": map[string]interface{}{
				"avg": map[string]interface{}{
					"field": "distance",
				},
			},
			"max_temp": map[string]interface{}{
				"max": map[string]interface{}{
					"field": "temperature",
				},
			},
			"min_temp": map[string]interface{}{
				"min": map[string]interface{}{
					"field": "temperature",
				},
			},
			"max_humidity": map[string]interface{}{
				"max": map[string]interface{}{
					"field": "humidity",
				},
			},
			"min_humidity": map[string]interface{}{
				"min": map[string]interface{}{
					"field": "humidity",
				},
			},
			"max_distance": map[string]interface{}{
				"max": map[string]interface{}{
					"field": "distance",
				},
			},
			"min_distance": map[string]interface{}{
				"min": map[string]interface{}{
					"field": "distance",
				},
			},
		},
		"size": 0, // 我们只需要聚合，不需要返回文档
	}

	must := query["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"].([]interface{})

	// 设备ID
	if req.DeviceID != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"device_id": req.DeviceID,
			},
		})
	}

	// 时间范围
	if req.StartDate != "" || req.EndDate != "" {
		rangeClause := map[string]interface{}{}
		if req.StartDate != "" {
			if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
				rangeClause["gte"] = t.Format(time.RFC3339)
			}
		}
		if req.EndDate != "" {
			if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
				rangeClause["lte"] = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second).Format(time.RFC3339)
			}
		}
		if len(rangeClause) > 0 {
			must = append(must, map[string]interface{}{
				"range": map[string]interface{}{
					"@timestamp": rangeClause,
				},
			})
		}
	}

	result, err := elasticsearch.ESClient.Search(query)
	if err != nil {
		log.Printf("[GetLogStats] 查询失败: %v", err)
		pkg.ErrorResponse(c, http.StatusInternalServerError, "查询失败")
		return
	}

	pkg.RightResponse(c, result, "查询成功")
}

// GetDeviceLogs 获取某设备的所有日志（分页）
func GetDeviceLogs(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		pkg.ErrorResponse(c, http.StatusBadRequest, "设备ID不能为空")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"match": map[string]interface{}{
							"device_id": deviceID,
						},
					},
				},
			},
		},
		"from": (page - 1) * pageSize,
		"size": pageSize,
		"sort": []interface{}{
			map[string]interface{}{
				"@timestamp": map[string]interface{}{
					"order": "desc",
				},
			},
		},
	}

	result, err := elasticsearch.ESClient.Search(query)
	if err != nil {
		log.Printf("[GetDeviceLogs] 查询失败: %v", err)
		pkg.ErrorResponse(c, http.StatusInternalServerError, "查询失败")
		return
	}

	pkg.RightResponse(c, result, "查询成功")
}
