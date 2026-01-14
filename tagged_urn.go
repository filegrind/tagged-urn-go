// Package capns provides the fundamental tagged URN system used across
// all FGND plugins and providers. It defines the formal structure for cap
// identifiers with flat tag-based naming, wildcard support, and specificity comparison.
package capns

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// TaggedUrn represents a tagged URN using flat, ordered tags
//
// Examples:
// - cap:op=generate;ext=pdf;out=std:binary.v1;target=thumbnail
// - cap:op=extract;target=metadata
// - cap:key="Value With Spaces"
type TaggedUrn struct {
	tags map[string]string
}

// TaggedUrnError represents errors that can occur during tagged URN operations
type TaggedUrnError struct {
	Code    int
	Message string
}

func (e *TaggedUrnError) Error() string {
	return e.Message
}

// Error codes for tagged URN operations
const (
	ErrorInvalidFormat         = 1
	ErrorEmptyTag              = 2
	ErrorInvalidCharacter      = 3
	ErrorInvalidTagFormat      = 4
	ErrorMissingCapPrefix      = 5
	ErrorDuplicateKey          = 6
	ErrorNumericKey            = 7
	ErrorUnterminatedQuote     = 8
	ErrorInvalidEscapeSequence = 9
)

// Parser states for state machine
type parseState int

const (
	stateExpectingKey parseState = iota
	stateInKey
	stateExpectingValue
	stateInUnquotedValue
	stateInQuotedValue
	stateInQuotedValueEscape
	stateExpectingSemiOrEnd
)

var numericPattern = regexp.MustCompile(`^[0-9]+$`)

// isValidKeyChar checks if a character is valid for a key
func isValidKeyChar(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' || c == '-' || c == '/' || c == ':' || c == '.'
}

// isValidUnquotedValueChar checks if a character is valid for an unquoted value
func isValidUnquotedValueChar(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' || c == '-' || c == '/' || c == ':' || c == '.' || c == '*'
}

// needsQuoting checks if a value needs quoting for serialization
func needsQuoting(value string) bool {
	for _, c := range value {
		if c == ';' || c == '=' || c == '"' || c == '\\' || c == ' ' || unicode.IsUpper(c) {
			return true
		}
	}
	return false
}

// quoteValue quotes a value for serialization
func quoteValue(value string) string {
	var result strings.Builder
	result.WriteRune('"')
	for _, c := range value {
		if c == '"' || c == '\\' {
			result.WriteRune('\\')
		}
		result.WriteRune(c)
	}
	result.WriteRune('"')
	return result.String()
}

// NewTaggedUrnFromString creates a tagged URN from a string
// Format: cap:key1=value1;key2=value2;... or cap:key1="value with spaces";key2=simple
// The "cap:" prefix is mandatory
// Trailing semicolons are optional and ignored
// Tags are automatically sorted alphabetically for canonical form
//
// Case handling:
// - Keys: Always normalized to lowercase
// - Unquoted values: Normalized to lowercase
// - Quoted values: Case preserved exactly as specified
func NewTaggedUrnFromString(s string) (*TaggedUrn, error) {
	if s == "" {
		return nil, &TaggedUrnError{
			Code:    ErrorInvalidFormat,
			Message: "tagged URN cannot be empty",
		}
	}

	// Check for "cap:" prefix (case-insensitive)
	if len(s) < 4 || !strings.EqualFold(s[:4], "cap:") {
		return nil, &TaggedUrnError{
			Code:    ErrorMissingCapPrefix,
			Message: "tagged URN must start with 'cap:'",
		}
	}

	tagsPart := s[4:]
	tags := make(map[string]string)

	// Handle empty tagged URN (cap: with no tags or just semicolon)
	if tagsPart == "" || tagsPart == ";" {
		return &TaggedUrn{tags: tags}, nil
	}

	state := stateExpectingKey
	var currentKey strings.Builder
	var currentValue strings.Builder
	chars := []rune(tagsPart)
	pos := 0

	finishTag := func() error {
		key := currentKey.String()
		value := currentValue.String()

		if key == "" {
			return &TaggedUrnError{
				Code:    ErrorEmptyTag,
				Message: "empty key",
			}
		}
		if value == "" {
			return &TaggedUrnError{
				Code:    ErrorEmptyTag,
				Message: fmt.Sprintf("empty value for key '%s'", key),
			}
		}

		// Check for duplicate keys
		if _, exists := tags[key]; exists {
			return &TaggedUrnError{
				Code:    ErrorDuplicateKey,
				Message: fmt.Sprintf("duplicate tag key: %s", key),
			}
		}

		// Validate key cannot be purely numeric
		if numericPattern.MatchString(key) {
			return &TaggedUrnError{
				Code:    ErrorNumericKey,
				Message: fmt.Sprintf("tag key cannot be purely numeric: %s", key),
			}
		}

		tags[key] = value
		currentKey.Reset()
		currentValue.Reset()
		return nil
	}

	for pos < len(chars) {
		c := chars[pos]

		switch state {
		case stateExpectingKey:
			if c == ';' {
				// Empty segment, skip
				pos++
				continue
			} else if isValidKeyChar(c) {
				currentKey.WriteRune(unicode.ToLower(c))
				state = stateInKey
			} else {
				return nil, &TaggedUrnError{
					Code:    ErrorInvalidCharacter,
					Message: fmt.Sprintf("invalid character '%c' at position %d", c, pos),
				}
			}

		case stateInKey:
			if c == '=' {
				if currentKey.Len() == 0 {
					return nil, &TaggedUrnError{
						Code:    ErrorEmptyTag,
						Message: "empty key",
					}
				}
				state = stateExpectingValue
			} else if isValidKeyChar(c) {
				currentKey.WriteRune(unicode.ToLower(c))
			} else {
				return nil, &TaggedUrnError{
					Code:    ErrorInvalidCharacter,
					Message: fmt.Sprintf("invalid character '%c' in key at position %d", c, pos),
				}
			}

		case stateExpectingValue:
			if c == '"' {
				state = stateInQuotedValue
			} else if c == ';' {
				return nil, &TaggedUrnError{
					Code:    ErrorEmptyTag,
					Message: fmt.Sprintf("empty value for key '%s'", currentKey.String()),
				}
			} else if isValidUnquotedValueChar(c) {
				currentValue.WriteRune(unicode.ToLower(c))
				state = stateInUnquotedValue
			} else {
				return nil, &TaggedUrnError{
					Code:    ErrorInvalidCharacter,
					Message: fmt.Sprintf("invalid character '%c' in value at position %d", c, pos),
				}
			}

		case stateInUnquotedValue:
			if c == ';' {
				if err := finishTag(); err != nil {
					return nil, err
				}
				state = stateExpectingKey
			} else if isValidUnquotedValueChar(c) {
				currentValue.WriteRune(unicode.ToLower(c))
			} else {
				return nil, &TaggedUrnError{
					Code:    ErrorInvalidCharacter,
					Message: fmt.Sprintf("invalid character '%c' in unquoted value at position %d", c, pos),
				}
			}

		case stateInQuotedValue:
			if c == '"' {
				state = stateExpectingSemiOrEnd
			} else if c == '\\' {
				state = stateInQuotedValueEscape
			} else {
				// Any character allowed in quoted value, preserve case
				currentValue.WriteRune(c)
			}

		case stateInQuotedValueEscape:
			if c == '"' || c == '\\' {
				currentValue.WriteRune(c)
				state = stateInQuotedValue
			} else {
				return nil, &TaggedUrnError{
					Code:    ErrorInvalidEscapeSequence,
					Message: fmt.Sprintf("invalid escape sequence at position %d (only \\\" and \\\\ allowed)", pos),
				}
			}

		case stateExpectingSemiOrEnd:
			if c == ';' {
				if err := finishTag(); err != nil {
					return nil, err
				}
				state = stateExpectingKey
			} else {
				return nil, &TaggedUrnError{
					Code:    ErrorInvalidCharacter,
					Message: fmt.Sprintf("expected ';' or end after quoted value, got '%c' at position %d", c, pos),
				}
			}
		}

		pos++
	}

	// Handle end of input
	switch state {
	case stateInUnquotedValue, stateExpectingSemiOrEnd:
		if err := finishTag(); err != nil {
			return nil, err
		}
	case stateExpectingKey:
		// Valid - trailing semicolon or empty input after prefix
	case stateInQuotedValue, stateInQuotedValueEscape:
		return nil, &TaggedUrnError{
			Code:    ErrorUnterminatedQuote,
			Message: fmt.Sprintf("unterminated quote at position %d", pos),
		}
	case stateInKey:
		return nil, &TaggedUrnError{
			Code:    ErrorInvalidTagFormat,
			Message: fmt.Sprintf("incomplete tag '%s'", currentKey.String()),
		}
	case stateExpectingValue:
		return nil, &TaggedUrnError{
			Code:    ErrorEmptyTag,
			Message: fmt.Sprintf("empty value for key '%s'", currentKey.String()),
		}
	}

	return &TaggedUrn{tags: tags}, nil
}

// NewTaggedUrnFromTags creates a tagged URN from tags
// Keys are normalized to lowercase; values are preserved as-is
func NewTaggedUrnFromTags(tags map[string]string) *TaggedUrn {
	result := make(map[string]string)
	for k, v := range tags {
		result[strings.ToLower(k)] = v
	}
	return &TaggedUrn{tags: result}
}

// GetTag returns the value of a specific tag
// Key is normalized to lowercase for lookup
func (c *TaggedUrn) GetTag(key string) (string, bool) {
	value, exists := c.tags[strings.ToLower(key)]
	return value, exists
}

// HasTag checks if this cap has a specific tag with a specific value
// Key is normalized to lowercase; value comparison is case-sensitive
func (c *TaggedUrn) HasTag(key, value string) bool {
	tagValue, exists := c.tags[strings.ToLower(key)]
	return exists && tagValue == value
}

// WithTag returns a new tagged URN with an added or updated tag
// Key is normalized to lowercase; value is preserved as-is
func (c *TaggedUrn) WithTag(key, value string) *TaggedUrn {
	newTags := make(map[string]string)
	for k, v := range c.tags {
		newTags[k] = v
	}
	newTags[strings.ToLower(key)] = value
	return &TaggedUrn{tags: newTags}
}

// WithoutTag returns a new tagged URN with a tag removed
// Key is normalized to lowercase for case-insensitive removal
func (c *TaggedUrn) WithoutTag(key string) *TaggedUrn {
	newTags := make(map[string]string)
	key = strings.ToLower(key)
	for k, v := range c.tags {
		if k != key {
			newTags[k] = v
		}
	}
	return &TaggedUrn{tags: newTags}
}

// Matches checks if this cap matches another based on tag compatibility
//
// A cap matches a request if:
// - For each tag in the request: cap has same value, wildcard (*), or missing tag
// - For each tag in the cap: if request is missing that tag, that's fine (cap is more specific)
// Missing tags are treated as wildcards (less specific, can handle any value).
func (c *TaggedUrn) Matches(request *TaggedUrn) bool {
	if request == nil {
		return true
	}

	// Check all tags that the request specifies
	for requestKey, requestValue := range request.tags {
		capValue, exists := c.tags[requestKey]
		if !exists {
			// Missing tag in cap is treated as wildcard - can handle any value
			continue
		}

		if capValue == "*" {
			// Cap has wildcard - can handle any value
			continue
		}

		if requestValue == "*" {
			// Request accepts any value - cap's specific value matches
			continue
		}

		if capValue != requestValue {
			// Cap has specific value that doesn't match request's specific value
			return false
		}
	}

	// If cap has additional specific tags that request doesn't specify, that's fine
	// The cap is just more specific than needed
	return true
}

// CanHandle checks if this cap can handle a request
func (c *TaggedUrn) CanHandle(request *TaggedUrn) bool {
	return c.Matches(request)
}

// Specificity returns the specificity score for cap matching
// More specific caps have higher scores and are preferred
func (c *TaggedUrn) Specificity() int {
	// Count non-wildcard tags
	count := 0
	for _, value := range c.tags {
		if value != "*" {
			count++
		}
	}
	return count
}

// IsMoreSpecificThan checks if this cap is more specific than another
func (c *TaggedUrn) IsMoreSpecificThan(other *TaggedUrn) bool {
	if other == nil {
		return true
	}

	// First check if they're compatible
	if !c.IsCompatibleWith(other) {
		return false
	}

	return c.Specificity() > other.Specificity()
}

// IsCompatibleWith checks if this cap is compatible with another
//
// Two caps are compatible if they can potentially match
// the same types of requests (considering wildcards and missing tags as wildcards)
func (c *TaggedUrn) IsCompatibleWith(other *TaggedUrn) bool {
	if other == nil {
		return true
	}

	// Get all unique tag keys from both caps
	allKeys := make(map[string]bool)
	for key := range c.tags {
		allKeys[key] = true
	}
	for key := range other.tags {
		allKeys[key] = true
	}

	for key := range allKeys {
		v1, exists1 := c.tags[key]
		v2, exists2 := other.tags[key]

		if exists1 && exists2 {
			// Both have the tag - they must match or one must be wildcard
			if v1 != "*" && v2 != "*" && v1 != v2 {
				return false
			}
		}
		// If only one has the tag, it's compatible (missing tag is wildcard)
	}

	return true
}

// WithWildcardTag returns a new cap with a specific tag set to wildcard
func (c *TaggedUrn) WithWildcardTag(key string) *TaggedUrn {
	if _, exists := c.tags[key]; exists {
		return c.WithTag(key, "*")
	}
	return c
}

// Subset returns a new cap with only specified tags
func (c *TaggedUrn) Subset(keys []string) *TaggedUrn {
	newTags := make(map[string]string)
	for _, key := range keys {
		if value, exists := c.tags[key]; exists {
			newTags[key] = value
		}
	}
	return &TaggedUrn{tags: newTags}
}

// Merge returns a new cap merged with another (other takes precedence for conflicts)
func (c *TaggedUrn) Merge(other *TaggedUrn) *TaggedUrn {
	newTags := make(map[string]string)
	for k, v := range c.tags {
		newTags[k] = v
	}
	for k, v := range other.tags {
		newTags[k] = v
	}
	return &TaggedUrn{tags: newTags}
}

// ToString returns the canonical string representation of this tagged URN
// Always includes "cap:" prefix
// Tags are sorted alphabetically for consistent representation
// No trailing semicolon in canonical form
// Values are quoted only when necessary (smart quoting)
func (c *TaggedUrn) ToString() string {
	if len(c.tags) == 0 {
		return "cap:"
	}

	// Sort keys for canonical representation
	keys := make([]string, 0, len(c.tags))
	for key := range c.tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Build tag string with smart quoting
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := c.tags[key]
		if needsQuoting(value) {
			parts = append(parts, fmt.Sprintf("%s=%s", key, quoteValue(value)))
		} else {
			parts = append(parts, fmt.Sprintf("%s=%s", key, value))
		}
	}

	tagsStr := strings.Join(parts, ";")
	return fmt.Sprintf("cap:%s", tagsStr)
}

// String implements the Stringer interface
func (c *TaggedUrn) String() string {
	return c.ToString()
}

// Equals checks if this tagged URN is equal to another
func (c *TaggedUrn) Equals(other *TaggedUrn) bool {
	if other == nil {
		return false
	}

	if len(c.tags) != len(other.tags) {
		return false
	}

	for key, value := range c.tags {
		otherValue, exists := other.tags[key]
		if !exists || value != otherValue {
			return false
		}
	}

	return true
}

// Hash returns a hash of this tagged URN
// Two equivalent tagged URNs will have the same hash
func (c *TaggedUrn) Hash() string {
	// Use canonical string representation for consistent hashing
	canonical := c.ToString()
	h := sha256.Sum256([]byte(canonical))
	return fmt.Sprintf("%x", h)
}

// MarshalJSON implements the json.Marshaler interface
func (c *TaggedUrn) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.ToString())
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (c *TaggedUrn) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("failed to unmarshal TaggedUrn: expected string, got: %s", string(data))
	}

	taggedUrn, err := NewTaggedUrnFromString(s)
	if err != nil {
		return err
	}

	c.tags = taggedUrn.tags
	return nil
}

// CapMatcher provides utility methods for matching caps
type CapMatcher struct{}

// FindBestMatch finds the most specific cap that can handle a request
func (m *CapMatcher) FindBestMatch(caps []*TaggedUrn, request *TaggedUrn) *TaggedUrn {
	var best *TaggedUrn
	bestSpecificity := -1

	for _, cap := range caps {
		if cap.CanHandle(request) {
			specificity := cap.Specificity()
			if specificity > bestSpecificity {
				best = cap
				bestSpecificity = specificity
			}
		}
	}

	return best
}

// FindAllMatches finds all caps that can handle a request, sorted by specificity
func (m *CapMatcher) FindAllMatches(caps []*TaggedUrn, request *TaggedUrn) []*TaggedUrn {
	var matches []*TaggedUrn

	for _, cap := range caps {
		if cap.CanHandle(request) {
			matches = append(matches, cap)
		}
	}

	// Sort by specificity (most specific first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Specificity() > matches[j].Specificity()
	})

	return matches
}

// AreCompatible checks if two cap sets are compatible
func (m *CapMatcher) AreCompatible(caps1, caps2 []*TaggedUrn) bool {
	for _, c1 := range caps1 {
		for _, c2 := range caps2 {
			if c1.IsCompatibleWith(c2) {
				return true
			}
		}
	}
	return false
}

// TaggedUrnBuilder provides a fluent builder interface for creating tagged URNs
type TaggedUrnBuilder struct {
	tags map[string]string
}

// NewTaggedUrnBuilder creates a new builder
func NewTaggedUrnBuilder() *TaggedUrnBuilder {
	return &TaggedUrnBuilder{
		tags: make(map[string]string),
	}
}

// Tag adds or updates a tag
// Key is normalized to lowercase; value is preserved as-is
func (b *TaggedUrnBuilder) Tag(key, value string) *TaggedUrnBuilder {
	b.tags[strings.ToLower(key)] = value
	return b
}

// Build creates the final TaggedUrn
func (b *TaggedUrnBuilder) Build() (*TaggedUrn, error) {
	if len(b.tags) == 0 {
		return nil, &TaggedUrnError{
			Code:    ErrorInvalidFormat,
			Message: "tagged URN cannot be empty",
		}
	}

	return &TaggedUrn{tags: b.tags}, nil
}
