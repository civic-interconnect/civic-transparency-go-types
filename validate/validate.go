package validate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/civic-interconnect/civic-transparency-go-types/types"
)

// MultiError is a tiny, allocation-light aggregator.
// Safe for concurrent use as long as each goroutine uses its own instance
type MultiError struct{ errs []error }

func MustProvenanceTag(t *types.ProvenanceTag) {
	if err := ValidateProvenanceTag(t); err != nil {
		panic(err)
	}
}
func MustSeries(s *types.Series) {
	if err := ValidateSeries(s); err != nil {
		panic(err)
	}
}

func (m *MultiError) Append(err error) {
	if err != nil {
		m.errs = append(m.errs, err)
	}
}
func (m *MultiError) Error() string {
	if len(m.errs) == 0 {
		return ""
	}
	var b strings.Builder
	for i, e := range m.errs {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(e.Error())
	}
	return b.String()
}

// Unwrap lets callers use errors.Is/As; Go 1.20+ errors.Join is efficient.
func (m *MultiError) Unwrap() error {
	if len(m.errs) == 0 {
		return nil
	}
	return errors.Join(m.errs...)
}

// NilOrError returns nil if empty, otherwise m.
func (m *MultiError) NilOrError() error {
	if len(m.errs) == 0 {
		return nil
	}
	return m
}

// ValidateProvenanceTag validates a single ProvenanceTag instance.
func ValidateProvenanceTag(t *types.ProvenanceTag) error {
	var me MultiError

	switch t.AcctAgeBucket {
	case types.AcctAge_0_7d, types.AcctAge_8_30d, types.AcctAge_1_6m, types.AcctAge_6_24m, types.AcctAge_24mPlus:
	default:
		me.Append(errors.New("invalid acct_age_bucket"))
	}
	switch t.AcctType {
	case types.AcctTypePerson, types.AcctTypeOrg, types.AcctTypeMedia,
		types.AcctTypePublicOfficial, types.AcctTypeUnverified, types.AcctTypeDeclaredAutomation:
	default:
		me.Append(errors.New("invalid acct_type"))
	}
	switch t.AutomationFlag {
	case types.AutomationManual, types.AutomationScheduled, types.AutomationAPICLIENT, types.AutomationDeclaredBot:
	default:
		me.Append(errors.New("invalid automation_flag"))
	}
	switch t.PostKind {
	case types.PostKindOriginal, types.PostKindReshare, types.PostKindQuote, types.PostKindReply:
	default:
		me.Append(errors.New("invalid post_kind"))
	}
	switch t.ClientFamily {
	case types.ClientWeb, types.ClientMobile, types.ClientThirdParty:
	default:
		me.Append(errors.New("invalid client_family"))
	}
	switch t.MediaProvenance {
	case types.MediaProvC2PA, types.MediaProvHash, types.MediaProvNone:
	default:
		me.Append(errors.New("invalid media_provenance"))
	}

	if !types.ReHex8.MatchString(string(t.DedupHash)) {
		me.Append(errors.New("dedup_hash must be 8 lowercase hex chars"))
	}
	if err := validateISO3166MaybeEmpty(t.OriginHint); err != nil {
		me.Append(err)
	}

	return me.NilOrError()
}

// ValidateSeries validates a Series instance and all nested Points.
func ValidateSeries(s *types.Series) error {
	var me MultiError

	if s.Topic == "" {
		me.Append(errors.New("topic must be non-empty"))
	}
	if s.GeneratedAt.IsZero() {
		me.Append(errors.New("generated_at must be set"))
	}
	if s.Interval != types.IntervalMinute {
		me.Append(errors.New("interval must be \"minute\""))
	}
	if len(s.Points) == 0 {
		me.Append(errors.New("series must contain at least one point"))
	}

	for i, p := range s.Points {
		if p.Volume < 0 {
			me.Append(fmt.Errorf("points[%d].volume must be ≥0", i))
		}
		if p.ReshareRatio < 0 || p.ReshareRatio > 1 {
			me.Append(fmt.Errorf("points[%d].reshare_ratio must be 0–1", i))
		}
		if p.RecycledContentRate < 0 || p.RecycledContentRate > 1 {
			me.Append(fmt.Errorf("points[%d].recycled_content_rate must be 0–1", i))
		}
		if p.CoordinationSignals.BurstScore < 0 || p.CoordinationSignals.BurstScore > 1 {
			me.Append(fmt.Errorf("points[%d].coordination_signals.burst_score must be 0–1", i))
		}
		if p.CoordinationSignals.SynchronyIndex < 0 || p.CoordinationSignals.SynchronyIndex > 1 {
			me.Append(fmt.Errorf("points[%d].coordination_signals.synchrony_index must be 0–1", i))
		}
		if p.CoordinationSignals.DuplicationClusters < 0 {
			me.Append(fmt.Errorf("points[%d].coordination_signals.duplication_clusters must be ≥0", i))
		}
	}

	return me.NilOrError()
}

// --- helpers ---

// validateISO3166MaybeEmpty accepts "" or a string that looks like ISO-3166
// country or country-subdivision code (e.g., "US" or "US-CA").
func validateISO3166MaybeEmpty(code string) error {
	if code == "" {
		return nil
	}
	if !types.ReISO3166.MatchString(code) {
		return fmt.Errorf("origin_hint/country must match ISO-3166 pattern (e.g., US or US-CA)")
	}
	return nil
}
