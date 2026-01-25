// Package taggedurn provides the fundamental tagged URN system with flat tag-based
// naming, pattern matching, and graded specificity comparison.
//
// Special pattern values:
//   - K=v: Must have key K with exact value v
//   - K=*: Must have key K with any value (presence required)
//   - K=!: Must NOT have key K (absence required)
//   - K=?: No constraint on key K (explicit don't-care)
//   - (missing): Same as K=? - no constraint
//
// Graded specificity scoring:
//   - Exact value (K=v): 3 points
//   - Must-have-any (K=*): 2 points
//   - Must-not-have (K=!): 1 point
//   - Unspecified (K=?) or missing: 0 points
package taggedurn

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// TaggedUrn represents a tagged URN using flat, ordered tags with a configurable prefix.
//
// Examples:
//   - cap:op=generate;ext=pdf;out=binary;target=thumbnail
//   - cap:format=*;debug=!  (format required, debug forbidden)
//   - myapp:key="Value With Spaces"
type TaggedUrn struct {
	prefix string
	tags   map[string]string
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
	ErrorMissingPrefix         = 5
	ErrorDuplicateKey          = 6
	ErrorNumericKey            = 7
	ErrorUnterminatedQuote     = 8
	ErrorInvalidEscapeSequence = 9
	ErrorEmptyPrefix           = 10
	ErrorPrefixMismatch        = 11
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
	return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' || c == '-' || c == '/' || c == ':' || c == '.' || c == '*' || c == '?' || c == '!'
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
// Format: prefix:key1=value1;key2=value2;... or prefix:key1="value with spaces";key2=simple
// The prefix is required and ends at the first colon
// Trailing semicolons are optional and ignored
// Tags are automatically sorted alphabetically for canonical form
//
// Case handling:
// - Prefix: Normalized to lowercase
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

	// Find the prefix (everything before the first colon)
	colonPos := strings.Index(s, ":")
	if colonPos == -1 {
		return nil, &TaggedUrnError{
			Code:    ErrorMissingPrefix,
			Message: "tagged URN must have a prefix followed by ':'",
		}
	}

	if colonPos == 0 {
		return nil, &TaggedUrnError{
			Code:    ErrorEmptyPrefix,
			Message: "tagged URN prefix cannot be empty",
		}
	}

	prefix := strings.ToLower(s[:colonPos])
	tagsPart := s[colonPos+1:]
	tags := make(map[string]string)

	// Handle empty tagged URN (prefix: with no tags or just semicolon)
	if tagsPart == "" || tagsPart == ";" {
		return &TaggedUrn{prefix: prefix, tags: tags}, nil
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
			} else if c == ';' {
				// Value-less tag: treat as wildcard
				if currentKey.Len() == 0 {
					return nil, &TaggedUrnError{
						Code:    ErrorEmptyTag,
						Message: "empty key",
					}
				}
				currentValue.WriteString("*")
				if err := finishTag(); err != nil {
					return nil, err
				}
				state = stateExpectingKey
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
		// Value-less tag at end: treat as wildcard
		if currentKey.Len() == 0 {
			return nil, &TaggedUrnError{
				Code:    ErrorEmptyTag,
				Message: "empty key",
			}
		}
		currentValue.WriteString("*")
		if err := finishTag(); err != nil {
			return nil, err
		}
	case stateExpectingValue:
		return nil, &TaggedUrnError{
			Code:    ErrorEmptyTag,
			Message: fmt.Sprintf("empty value for key '%s'", currentKey.String()),
		}
	}

	return &TaggedUrn{prefix: prefix, tags: tags}, nil
}

// NewTaggedUrnFromTags creates a tagged URN from tags with a specified prefix (required)
// Keys are normalized to lowercase; values are preserved as-is
func NewTaggedUrnFromTags(prefix string, tags map[string]string) *TaggedUrn {
	result := make(map[string]string)
	for k, v := range tags {
		result[strings.ToLower(k)] = v
	}
	return &TaggedUrn{prefix: strings.ToLower(prefix), tags: result}
}

// Empty creates an empty tagged URN with the specified prefix (required)
func Empty(prefix string) *TaggedUrn {
	return &TaggedUrn{prefix: strings.ToLower(prefix), tags: make(map[string]string)}
}

// GetPrefix returns the prefix of this tagged URN
func (c *TaggedUrn) GetPrefix() string {
	return c.prefix
}

// GetTag returns the value of a specific tag
// Key is normalized to lowercase for lookup
func (c *TaggedUrn) GetTag(key string) (string, bool) {
	value, exists := c.tags[strings.ToLower(key)]
	return value, exists
}

// AllTags returns a copy of all tags in this URN
func (c *TaggedUrn) AllTags() map[string]string {
	result := make(map[string]string, len(c.tags))
	for k, v := range c.tags {
		result[k] = v
	}
	return result
}

// HasTag checks if this URN has a specific tag with a specific value
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
	return &TaggedUrn{prefix: c.prefix, tags: newTags}
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
	return &TaggedUrn{prefix: c.prefix, tags: newTags}
}

// Matches checks if this URN (instance) matches a pattern based on tag compatibility
//
// IMPORTANT: Both URNs must have the same prefix. Comparing URNs with
// different prefixes is a programming error and will return an error.
//
// Per-tag matching semantics:
// | Pattern Form | Interpretation              | Instance Missing | Instance = v | Instance = x≠v |
// |--------------|-----------------------------|--------------------|--------------|----------------|
// | (no entry)   | no constraint               | OK match           | OK match     | OK match       |
// | K=?          | no constraint (explicit)    | OK                 | OK           | OK             |
// | K=!          | must-not-have               | OK                 | NO           | NO             |
// | K=*          | must-have, any value        | NO                 | OK           | OK             |
// | K=v          | must-have, exact value      | NO                 | OK           | NO             |
//
// Special values work symmetrically on both instance and pattern sides.
func (c *TaggedUrn) Matches(pattern *TaggedUrn) (bool, error) {
	if pattern == nil {
		return false, &TaggedUrnError{
			Code:    ErrorInvalidFormat,
			Message: "cannot match against nil pattern",
		}
	}

	// First check prefix - must match exactly
	if c.prefix != pattern.prefix {
		return false, &TaggedUrnError{
			Code:    ErrorPrefixMismatch,
			Message: fmt.Sprintf("cannot compare URNs with different prefixes: '%s' vs '%s'", c.prefix, pattern.prefix),
		}
	}

	// Collect all keys from both instance and pattern
	allKeys := make(map[string]bool)
	for key := range c.tags {
		allKeys[key] = true
	}
	for key := range pattern.tags {
		allKeys[key] = true
	}

	for key := range allKeys {
		inst, instExists := c.tags[key]
		patt, pattExists := pattern.tags[key]

		var instVal, pattVal *string
		if instExists {
			instVal = &inst
		}
		if pattExists {
			pattVal = &patt
		}

		if !valuesMatch(instVal, pattVal) {
			return false, nil
		}
	}
	return true, nil
}

// valuesMatch checks if instance value matches pattern constraint
//
// Full cross-product truth table:
// | Instance | Pattern | Match? | Reason |
// |----------|---------|--------|--------|
// | (none)   | (none)  | OK     | No constraint either side |
// | (none)   | K=?     | OK     | Pattern doesn't care |
// | (none)   | K=!     | OK     | Pattern wants absent, it is |
// | (none)   | K=*     | NO     | Pattern wants present |
// | (none)   | K=v     | NO     | Pattern wants exact value |
// | K=?      | (any)   | OK     | Instance doesn't care |
// | K=!      | (none)  | OK     | Symmetric: absent |
// | K=!      | K=?     | OK     | Pattern doesn't care |
// | K=!      | K=!     | OK     | Both want absent |
// | K=!      | K=*     | NO     | Conflict: absent vs present |
// | K=!      | K=v     | NO     | Conflict: absent vs value |
// | K=*      | (none)  | OK     | Pattern has no constraint |
// | K=*      | K=?     | OK     | Pattern doesn't care |
// | K=*      | K=!     | NO     | Conflict: present vs absent |
// | K=*      | K=*     | OK     | Both accept any presence |
// | K=*      | K=v     | OK     | Instance accepts any, v is fine |
// | K=v      | (none)  | OK     | Pattern has no constraint |
// | K=v      | K=?     | OK     | Pattern doesn't care |
// | K=v      | K=!     | NO     | Conflict: value vs absent |
// | K=v      | K=*     | OK     | Pattern wants any, v satisfies |
// | K=v      | K=v     | OK     | Exact match |
// | K=v      | K=w     | NO     | Value mismatch (v≠w) |
func valuesMatch(inst, patt *string) bool {
	// Pattern has no constraint (no entry or explicit ?)
	if patt == nil || *patt == "?" {
		return true
	}

	// Instance doesn't care (explicit ?)
	if inst != nil && *inst == "?" {
		return true
	}

	// Pattern: must-not-have (!)
	if *patt == "!" {
		if inst == nil {
			return true // Instance absent, pattern wants absent
		}
		if *inst == "!" {
			return true // Both say absent
		}
		return false // Instance has value, pattern wants absent
	}

	// Instance: must-not-have conflicts with pattern wanting value
	if inst != nil && *inst == "!" {
		return false // Conflict: absent vs value or present
	}

	// Pattern: must-have-any (*)
	if *patt == "*" {
		if inst == nil {
			return false // Instance missing, pattern wants present
		}
		return true // Instance has value, pattern wants any
	}

	// Pattern: exact value
	if inst == nil {
		return false // Instance missing, pattern wants value
	}
	if *inst == "*" {
		return true // Instance accepts any, pattern's value is fine
	}
	return *inst == *patt // Both have values, must match exactly
}

// CanHandle checks if this URN can handle a request
func (c *TaggedUrn) CanHandle(request *TaggedUrn) (bool, error) {
	return c.Matches(request)
}

// Specificity returns the specificity score for URN matching
// More specific URNs have higher scores and are preferred
// Graded scoring:
// - K=v (exact value): 3 points (most specific)
// - K=* (must-have-any): 2 points
// - K=! (must-not-have): 1 point
// - K=? (unspecified): 0 points (least specific)
func (c *TaggedUrn) Specificity() int {
	score := 0
	for _, value := range c.tags {
		switch value {
		case "?":
			score += 0
		case "!":
			score += 1
		case "*":
			score += 2
		default:
			score += 3 // exact value
		}
	}
	return score
}

// SpecificityTuple returns specificity as a tuple for tie-breaking
// Returns (exact_count, must_have_any_count, must_not_count)
// Compare tuples lexicographically when sum scores are equal
func (c *TaggedUrn) SpecificityTuple() (int, int, int) {
	exact := 0
	mustHaveAny := 0
	mustNot := 0
	for _, value := range c.tags {
		switch value {
		case "?":
			// 0 points, not counted
		case "!":
			mustNot++
		case "*":
			mustHaveAny++
		default:
			exact++
		}
	}
	return exact, mustHaveAny, mustNot
}

// IsMoreSpecificThan checks if this URN is more specific than another
func (c *TaggedUrn) IsMoreSpecificThan(other *TaggedUrn) (bool, error) {
	if other == nil {
		return false, &TaggedUrnError{
			Code:    ErrorInvalidFormat,
			Message: "cannot compare against nil URN",
		}
	}

	// First check prefix
	if c.prefix != other.prefix {
		return false, &TaggedUrnError{
			Code:    ErrorPrefixMismatch,
			Message: fmt.Sprintf("cannot compare URNs with different prefixes: '%s' vs '%s'", c.prefix, other.prefix),
		}
	}

	// Then check if they're compatible
	compatible, err := c.IsCompatibleWith(other)
	if err != nil {
		return false, err
	}
	if !compatible {
		return false, nil
	}

	return c.Specificity() > other.Specificity(), nil
}

// IsCompatibleWith checks if this URN is compatible with another
//
// Two URNs are compatible if they have the same prefix and can potentially match
// the same instances (i.e., there exists at least one instance that both patterns accept)
//
// Compatibility rules:
// - K=v and K=w (v≠w): NOT compatible (no instance can match both exact values)
// - K=! and K=v/K=*: NOT compatible (one requires absent, other requires present)
// - K=v and K=*: compatible (instance with K=v matches both)
// - K=? is compatible with anything (no constraint)
// - Missing entry is compatible with anything (no constraint)
func (c *TaggedUrn) IsCompatibleWith(other *TaggedUrn) (bool, error) {
	if other == nil {
		return false, &TaggedUrnError{
			Code:    ErrorInvalidFormat,
			Message: "cannot check compatibility with nil URN",
		}
	}

	// First check prefix
	if c.prefix != other.prefix {
		return false, &TaggedUrnError{
			Code:    ErrorPrefixMismatch,
			Message: fmt.Sprintf("cannot compare URNs with different prefixes: '%s' vs '%s'", c.prefix, other.prefix),
		}
	}

	// Get all unique tag keys from both URNs
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

		var val1, val2 *string
		if exists1 {
			val1 = &v1
		}
		if exists2 {
			val2 = &v2
		}

		if !valuesCompatible(val1, val2) {
			return false, nil
		}
	}

	return true, nil
}

// valuesCompatible checks if two pattern values are compatible (could match the same instance)
func valuesCompatible(v1, v2 *string) bool {
	// Either missing or ? means no constraint - compatible with anything
	if v1 == nil || v2 == nil {
		return true
	}
	if *v1 == "?" || *v2 == "?" {
		return true
	}

	// Both are ! - compatible (both want absent)
	if *v1 == "!" && *v2 == "!" {
		return true
	}

	// One is ! and other is value or * - NOT compatible
	if *v1 == "!" || *v2 == "!" {
		return false
	}

	// Both are * - compatible
	if *v1 == "*" && *v2 == "*" {
		return true
	}

	// One is * and other is value - compatible (value matches *)
	if *v1 == "*" || *v2 == "*" {
		return true
	}

	// Both are specific values - must be equal
	return *v1 == *v2
}

// WithWildcardTag returns a new URN with a specific tag set to wildcard
func (c *TaggedUrn) WithWildcardTag(key string) *TaggedUrn {
	if _, exists := c.tags[key]; exists {
		return c.WithTag(key, "*")
	}
	return c
}

// Subset returns a new URN with only specified tags
func (c *TaggedUrn) Subset(keys []string) *TaggedUrn {
	newTags := make(map[string]string)
	for _, key := range keys {
		if value, exists := c.tags[key]; exists {
			newTags[key] = value
		}
	}
	return &TaggedUrn{prefix: c.prefix, tags: newTags}
}

// Merge returns a new URN merged with another (other takes precedence for conflicts)
// Both must have the same prefix
func (c *TaggedUrn) Merge(other *TaggedUrn) (*TaggedUrn, error) {
	if other == nil {
		return nil, &TaggedUrnError{
			Code:    ErrorInvalidFormat,
			Message: "cannot merge with nil URN",
		}
	}

	if c.prefix != other.prefix {
		return nil, &TaggedUrnError{
			Code:    ErrorPrefixMismatch,
			Message: fmt.Sprintf("cannot merge URNs with different prefixes: '%s' vs '%s'", c.prefix, other.prefix),
		}
	}

	newTags := make(map[string]string)
	for k, v := range c.tags {
		newTags[k] = v
	}
	for k, v := range other.tags {
		newTags[k] = v
	}
	return &TaggedUrn{prefix: c.prefix, tags: newTags}, nil
}

// ToString returns the canonical string representation of this tagged URN
// Uses the stored prefix
// Tags are sorted alphabetically for consistent representation
// No trailing semicolon in canonical form
// Values are quoted only when necessary (smart quoting)
// Special value serialization:
// - * (must-have-any): serialized as value-less tag (just the key)
// - ? (unspecified): serialized as key=?
// - ! (must-not-have): serialized as key=!
func (c *TaggedUrn) ToString() string {
	if len(c.tags) == 0 {
		return fmt.Sprintf("%s:", c.prefix)
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
		switch value {
		case "*":
			// Valueless sugar: key
			parts = append(parts, key)
		case "?":
			// Explicit: key=?
			parts = append(parts, fmt.Sprintf("%s=?", key))
		case "!":
			// Explicit: key=!
			parts = append(parts, fmt.Sprintf("%s=!", key))
		default:
			if needsQuoting(value) {
				parts = append(parts, fmt.Sprintf("%s=%s", key, quoteValue(value)))
			} else {
				parts = append(parts, fmt.Sprintf("%s=%s", key, value))
			}
		}
	}

	tagsStr := strings.Join(parts, ";")
	return fmt.Sprintf("%s:%s", c.prefix, tagsStr)
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

	if c.prefix != other.prefix {
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

	c.prefix = taggedUrn.prefix
	c.tags = taggedUrn.tags
	return nil
}

// UrnMatcher provides utility methods for matching URNs
type UrnMatcher struct{}

// FindBestMatch finds the most specific URN that can handle a request
// All URNs must have the same prefix as the request
func (m *UrnMatcher) FindBestMatch(urns []*TaggedUrn, request *TaggedUrn) (*TaggedUrn, error) {
	var best *TaggedUrn
	bestSpecificity := -1

	for _, urn := range urns {
		canHandle, err := urn.CanHandle(request)
		if err != nil {
			return nil, err
		}
		if canHandle {
			specificity := urn.Specificity()
			if specificity > bestSpecificity {
				best = urn
				bestSpecificity = specificity
			}
		}
	}

	return best, nil
}

// FindAllMatches finds all URNs that can handle a request, sorted by specificity
// All URNs must have the same prefix as the request
func (m *UrnMatcher) FindAllMatches(urns []*TaggedUrn, request *TaggedUrn) ([]*TaggedUrn, error) {
	var matches []*TaggedUrn

	for _, urn := range urns {
		canHandle, err := urn.CanHandle(request)
		if err != nil {
			return nil, err
		}
		if canHandle {
			matches = append(matches, urn)
		}
	}

	// Sort by specificity (most specific first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Specificity() > matches[j].Specificity()
	})

	return matches, nil
}

// AreCompatible checks if two URN sets are compatible
// All URNs in both sets must have the same prefix
func (m *UrnMatcher) AreCompatible(urns1, urns2 []*TaggedUrn) (bool, error) {
	for _, u1 := range urns1 {
		for _, u2 := range urns2 {
			compatible, err := u1.IsCompatibleWith(u2)
			if err != nil {
				return false, err
			}
			if compatible {
				return true, nil
			}
		}
	}
	return false, nil
}

// TaggedUrnBuilder provides a fluent builder interface for creating tagged URNs
type TaggedUrnBuilder struct {
	prefix string
	tags   map[string]string
}

// NewTaggedUrnBuilder creates a new builder with a specified prefix (required)
func NewTaggedUrnBuilder(prefix string) *TaggedUrnBuilder {
	return &TaggedUrnBuilder{
		prefix: strings.ToLower(prefix),
		tags:   make(map[string]string),
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

	return &TaggedUrn{prefix: b.prefix, tags: b.tags}, nil
}

// BuildAllowEmpty creates the final TaggedUrn, allowing empty tags
func (b *TaggedUrnBuilder) BuildAllowEmpty() *TaggedUrn {
	return &TaggedUrn{prefix: b.prefix, tags: b.tags}
}
