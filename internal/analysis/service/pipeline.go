package service

import "context"

type Analyzer interface {
	Analyze(ctx context.Context, tenantID, gameID string) (int, error)
}

type MetricCalculator interface {
	Recalculate(ctx context.Context, tenantID, gameID string) (int, error)
}

type Result struct {
	CalculatedRows      int `json:"calculated_rows"`
	BusinessFindings    int `json:"business_findings"`
	AttributionFindings int `json:"attribution_findings"`
	CreativeFindings    int `json:"creative_findings"`
}

type Pipeline struct {
	metrics     MetricCalculator
	business    Analyzer
	attribution Analyzer
	creative    Analyzer
}

func NewPipeline(metrics MetricCalculator, business, attribution, creative Analyzer) *Pipeline {
	return &Pipeline{metrics: metrics, business: business, attribution: attribution, creative: creative}
}

func (p *Pipeline) Run(ctx context.Context, tenantID, gameID string) (*Result, error) {
	result := &Result{}
	var err error
	if result.CalculatedRows, err = p.metrics.Recalculate(ctx, tenantID, gameID); err != nil {
		return nil, err
	}
	if result.BusinessFindings, err = p.business.Analyze(ctx, tenantID, gameID); err != nil {
		return nil, err
	}
	if result.AttributionFindings, err = p.attribution.Analyze(ctx, tenantID, gameID); err != nil {
		return nil, err
	}
	if result.CreativeFindings, err = p.creative.Analyze(ctx, tenantID, gameID); err != nil {
		return nil, err
	}
	return result, nil
}
