package collector

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

type HotSpotCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newHotSpotCollector() routerOSCollector {
	c := &HotSpotCollector{}
	c.init()
	return c
}

func (c *HotSpotCollector) init() {
	c.props = []string{"user", "uptime", "bytes-in", "bytes-out", "packets-in", "packets-out"}

	labelNames := []string{"name", "address", "user", "comment"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[1:] {
		c.descriptions[p] = descriptionForPropertyName("user", p, labelNames)
	}
}

func (c *HotSpotCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *HotSpotCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *HotSpotCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/ip/hotspot/active/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching interface metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *HotSpotCollector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	user := re.Map["user"]
	name := re.Map["name"]
	comment := re.Map["comment"]

	for _, p := range c.props[2:] {
		c.collectMetricForProperty(p, user, name, comment, re, ctx)
	}
}

func (c *HotSpotCollector) collectMetricForProperty(property, user, name, comment string, re *proto.Sentence, ctx *collectorContext) {
	desc := c.descriptions[property]
	if value := re.Map[property]; value != "" {
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.WithFields(log.Fields{
				"device":    ctx.device.Name,
				"name": name,
				"user": user,
				"property":  property,
				"value":     value,
				"error":     err,
			}).Error("error parsing hotspot metric value")
			return
		}
		ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v, ctx.device.Name, ctx.device.Address, user, comment)
	}
}
