package capns

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaggedUrnCreation(t *testing.T) {
	taggedUrn, err := NewTaggedUrnFromString("cap:op=transform;format=json;type=data_processing")

	assert.NoError(t, err)
	assert.NotNil(t, taggedUrn)

	capType, exists := taggedUrn.GetTag("type")
	assert.True(t, exists)
	assert.Equal(t, "data_processing", capType)

	op, exists := taggedUrn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "transform", op)

	format, exists := taggedUrn.GetTag("format")
	assert.True(t, exists)
	assert.Equal(t, "json", format)
}

func TestCanonicalStringFormat(t *testing.T) {
	taggedUrn, err := NewTaggedUrnFromString("cap:op=generate;target=thumbnail;ext=pdf")
	require.NoError(t, err)

	// Should be sorted alphabetically and have no trailing semicolon in canonical form
	assert.Equal(t, "cap:ext=pdf;op=generate;target=thumbnail", taggedUrn.ToString())
}

func TestCapPrefixRequired(t *testing.T) {
	// Missing cap: prefix should fail
	taggedUrn, err := NewTaggedUrnFromString("op=generate;ext=pdf")
	assert.Nil(t, taggedUrn)
	assert.Error(t, err)
	assert.Equal(t, ErrorMissingCapPrefix, err.(*TaggedUrnError).Code)

	// Valid cap: prefix should work
	taggedUrn, err = NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	assert.NoError(t, err)
	assert.NotNil(t, taggedUrn)
	op, exists := taggedUrn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)

	// Case-insensitive prefix
	taggedUrn, err = NewTaggedUrnFromString("CAP:op=generate")
	assert.NoError(t, err)
	op, exists = taggedUrn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)
}

func TestTrailingSemicolonEquivalence(t *testing.T) {
	// Both with and without trailing semicolon should be equivalent
	cap1, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	cap2, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;")
	require.NoError(t, err)

	// They should be equal
	assert.True(t, cap1.Equals(cap2))

	// They should have same hash
	assert.Equal(t, cap1.Hash(), cap2.Hash())

	// They should have same string representation (canonical form)
	assert.Equal(t, cap1.ToString(), cap2.ToString())

	// They should match each other
	assert.True(t, cap1.Matches(cap2))
	assert.True(t, cap2.Matches(cap1))
}

func TestInvalidTaggedUrn(t *testing.T) {
	taggedUrn, err := NewTaggedUrnFromString("")

	assert.Nil(t, taggedUrn)
	assert.Error(t, err)
	assert.Equal(t, ErrorInvalidFormat, err.(*TaggedUrnError).Code)
}

func TestInvalidTagFormat(t *testing.T) {
	taggedUrn, err := NewTaggedUrnFromString("cap:invalid_tag")

	assert.Nil(t, taggedUrn)
	assert.Error(t, err)
	assert.Equal(t, ErrorInvalidTagFormat, err.(*TaggedUrnError).Code)
}

func TestInvalidCharacters(t *testing.T) {
	taggedUrn, err := NewTaggedUrnFromString("cap:type@invalid=value")

	assert.Nil(t, taggedUrn)
	assert.Error(t, err)
	assert.Equal(t, ErrorInvalidCharacter, err.(*TaggedUrnError).Code)
}

func TestTagMatching(t *testing.T) {
	cap, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;target=thumbnail")
	require.NoError(t, err)

	// Exact match
	request1, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;target=thumbnail")
	require.NoError(t, err)
	assert.True(t, cap.Matches(request1))

	// Subset match
	request2, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)
	assert.True(t, cap.Matches(request2))

	// Wildcard match
	request3, err := NewTaggedUrnFromString("cap:ext=*")
	require.NoError(t, err)
	assert.True(t, cap.Matches(request3))

	// No match - conflicting value
	request4, err := NewTaggedUrnFromString("cap:op=extract")
	require.NoError(t, err)
	assert.False(t, cap.Matches(request4))
}

func TestMissingTagHandling(t *testing.T) {
	cap, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	// Request with missing tag should fail if specific value required
	request1, err := NewTaggedUrnFromString("cap:ext=pdf")
	require.NoError(t, err)
	assert.True(t, cap.Matches(request1)) // cap missing format tag = wildcard, can handle any format

	// But cap with extra tags can match subset requests
	cap2, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)
	request2, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)
	assert.True(t, cap2.Matches(request2))
}

func TestSpecificity(t *testing.T) {
	cap1, err := NewTaggedUrnFromString("cap:op=*")
	require.NoError(t, err)

	cap2, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	cap3, err := NewTaggedUrnFromString("cap:op=*;ext=pdf")
	require.NoError(t, err)

	assert.Equal(t, 0, cap1.Specificity()) // wildcard doesn't count
	assert.Equal(t, 1, cap2.Specificity())
	assert.Equal(t, 1, cap3.Specificity()) // only ext=pdf counts, op=* doesn't count

	assert.True(t, cap2.IsMoreSpecificThan(cap1))
}

func TestCompatibility(t *testing.T) {
	cap1, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	cap2, err := NewTaggedUrnFromString("cap:op=generate;format=*")
	require.NoError(t, err)

	cap3, err := NewTaggedUrnFromString("cap:op=extract;ext=pdf")
	require.NoError(t, err)

	assert.True(t, cap1.IsCompatibleWith(cap2))
	assert.True(t, cap2.IsCompatibleWith(cap1))
	assert.False(t, cap1.IsCompatibleWith(cap3))

	// Missing tags are treated as wildcards for compatibility
	cap4, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)
	assert.True(t, cap1.IsCompatibleWith(cap4))
	assert.True(t, cap4.IsCompatibleWith(cap1))
}

func TestConvenienceMethods(t *testing.T) {
	cap, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;output=binary;target=thumbnail")
	require.NoError(t, err)

	op, exists := cap.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)

	target, exists := cap.GetTag("target")
	assert.True(t, exists)
	assert.Equal(t, "thumbnail", target)

	format, exists := cap.GetTag("ext")
	assert.True(t, exists)
	assert.Equal(t, "pdf", format)

	output, exists := cap.GetTag("output")
	assert.True(t, exists)
	assert.Equal(t, "binary", output)
}

func TestBuilder(t *testing.T) {
	cap, err := NewTaggedUrnBuilder().
		Tag("op", "generate").
		Tag("target", "thumbnail").
		Tag("ext", "pdf").
		Tag("output", "binary").
		Build()
	require.NoError(t, err)

	op, exists := cap.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)

	output, exists := cap.GetTag("output")
	assert.True(t, exists)
	assert.Equal(t, "binary", output)
}

func TestWithTag(t *testing.T) {
	original, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	modified := original.WithTag("ext", "pdf")

	assert.Equal(t, "cap:ext=pdf;op=generate", modified.ToString())

	// Original should be unchanged
	assert.Equal(t, "cap:op=generate", original.ToString())
}

func TestWithoutTag(t *testing.T) {
	original, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	modified := original.WithoutTag("ext")

	assert.Equal(t, "cap:op=generate", modified.ToString())

	// Original should be unchanged
	assert.Equal(t, "cap:ext=pdf;op=generate", original.ToString())
}

func TestWildcardTag(t *testing.T) {
	cap, err := NewTaggedUrnFromString("cap:ext=pdf")
	require.NoError(t, err)

	wildcarded := cap.WithWildcardTag("ext")

	assert.Equal(t, "cap:ext=*", wildcarded.ToString())

	// Test that wildcarded cap can match more requests
	request, err := NewTaggedUrnFromString("cap:ext=jpg")
	require.NoError(t, err)
	assert.False(t, cap.Matches(request))

	wildcardRequest, err := NewTaggedUrnFromString("cap:ext=*")
	require.NoError(t, err)
	assert.True(t, wildcarded.Matches(wildcardRequest))
}

func TestSubset(t *testing.T) {
	cap, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;output=binary;target=thumbnail;")
	require.NoError(t, err)

	subset := cap.Subset([]string{"type", "ext"})

	assert.Equal(t, "cap:ext=pdf", subset.ToString())
}

func TestMerge(t *testing.T) {
	cap1, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	cap2, err := NewTaggedUrnFromString("cap:ext=pdf;output=binary")
	require.NoError(t, err)

	merged := cap1.Merge(cap2)

	assert.Equal(t, "cap:ext=pdf;op=generate;output=binary", merged.ToString())
}

func TestEquality(t *testing.T) {
	cap1, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	cap2, err := NewTaggedUrnFromString("cap:op=generate") // different order
	require.NoError(t, err)

	cap3, err := NewTaggedUrnFromString("cap:op=generate;type=image")
	require.NoError(t, err)

	assert.True(t, cap1.Equals(cap2)) // order doesn't matter
	assert.False(t, cap1.Equals(cap3))
}

func TestCapMatcher(t *testing.T) {
	matcher := &CapMatcher{}

	caps := []*TaggedUrn{}

	cap1, err := NewTaggedUrnFromString("cap:op=*")
	require.NoError(t, err)
	caps = append(caps, cap1)

	cap2, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)
	caps = append(caps, cap2)

	cap3, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)
	caps = append(caps, cap3)

	request, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	best := matcher.FindBestMatch(caps, request)
	require.NotNil(t, best)

	// Most specific cap that can handle the request
	assert.Equal(t, "cap:ext=pdf;op=generate", best.ToString())
}

func TestJSONSerialization(t *testing.T) {
	original, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	data, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.NotNil(t, data)

	var decoded TaggedUrn
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.True(t, original.Equals(&decoded))
}

func TestUnquotedValuesLowercased(t *testing.T) {
	// Unquoted values are normalized to lowercase
	cap, err := NewTaggedUrnFromString("cap:OP=Generate;EXT=PDF;Target=Thumbnail;")
	require.NoError(t, err)

	// Keys are always lowercase
	op, exists := cap.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)

	ext, exists := cap.GetTag("ext")
	assert.True(t, exists)
	assert.Equal(t, "pdf", ext)

	target, exists := cap.GetTag("target")
	assert.True(t, exists)
	assert.Equal(t, "thumbnail", target)

	// Key lookup is case-insensitive
	op2, exists := cap.GetTag("OP")
	assert.True(t, exists)
	assert.Equal(t, "generate", op2)

	// Both URNs parse to same lowercase values
	cap2, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;target=thumbnail;")
	require.NoError(t, err)
	assert.Equal(t, cap.ToString(), cap2.ToString())
	assert.True(t, cap.Equals(cap2))
}

func TestQuotedValuesPreserveCase(t *testing.T) {
	// Quoted values preserve their case
	cap, err := NewTaggedUrnFromString(`cap:key="Value With Spaces"`)
	require.NoError(t, err)
	value, exists := cap.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "Value With Spaces", value)

	// Key is still lowercase
	cap2, err := NewTaggedUrnFromString(`cap:KEY="Value With Spaces"`)
	require.NoError(t, err)
	value2, exists := cap2.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "Value With Spaces", value2)

	// Unquoted vs quoted case difference
	unquoted, err := NewTaggedUrnFromString("cap:key=UPPERCASE")
	require.NoError(t, err)
	quoted, err := NewTaggedUrnFromString(`cap:key="UPPERCASE"`)
	require.NoError(t, err)

	unquotedVal, _ := unquoted.GetTag("key")
	quotedVal, _ := quoted.GetTag("key")
	assert.Equal(t, "uppercase", unquotedVal) // lowercase
	assert.Equal(t, "UPPERCASE", quotedVal)   // preserved
	assert.False(t, unquoted.Equals(quoted))  // NOT equal
}

func TestQuotedValueSpecialChars(t *testing.T) {
	// Semicolons in quoted values
	cap, err := NewTaggedUrnFromString(`cap:key="value;with;semicolons"`)
	require.NoError(t, err)
	value, exists := cap.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "value;with;semicolons", value)

	// Equals in quoted values
	cap2, err := NewTaggedUrnFromString(`cap:key="value=with=equals"`)
	require.NoError(t, err)
	value2, exists := cap2.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "value=with=equals", value2)

	// Spaces in quoted values
	cap3, err := NewTaggedUrnFromString(`cap:key="hello world"`)
	require.NoError(t, err)
	value3, exists := cap3.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "hello world", value3)
}

func TestQuotedValueEscapeSequences(t *testing.T) {
	// Escaped quotes
	cap, err := NewTaggedUrnFromString(`cap:key="value\"quoted\""`)
	require.NoError(t, err)
	value, exists := cap.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, `value"quoted"`, value)

	// Escaped backslashes
	cap2, err := NewTaggedUrnFromString(`cap:key="path\\file"`)
	require.NoError(t, err)
	value2, exists := cap2.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, `path\file`, value2)

	// Mixed escapes
	cap3, err := NewTaggedUrnFromString(`cap:key="say \"hello\\world\""`)
	require.NoError(t, err)
	value3, exists := cap3.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, `say "hello\world"`, value3)
}

func TestMixedQuotedUnquoted(t *testing.T) {
	cap, err := NewTaggedUrnFromString(`cap:a="Quoted";b=simple`)
	require.NoError(t, err)

	a, exists := cap.GetTag("a")
	assert.True(t, exists)
	assert.Equal(t, "Quoted", a)

	b, exists := cap.GetTag("b")
	assert.True(t, exists)
	assert.Equal(t, "simple", b)
}

func TestUnterminatedQuoteError(t *testing.T) {
	cap, err := NewTaggedUrnFromString(`cap:key="unterminated`)
	assert.Nil(t, cap)
	assert.Error(t, err)
	capError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorUnterminatedQuote, capError.Code)
}

func TestInvalidEscapeSequenceError(t *testing.T) {
	cap, err := NewTaggedUrnFromString(`cap:key="bad\n"`)
	assert.Nil(t, cap)
	assert.Error(t, err)
	capError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorInvalidEscapeSequence, capError.Code)

	// Invalid escape at end
	cap2, err := NewTaggedUrnFromString(`cap:key="bad\x"`)
	assert.Nil(t, cap2)
	assert.Error(t, err)
	capError2, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorInvalidEscapeSequence, capError2.Code)
}

func TestSerializationSmartQuoting(t *testing.T) {
	// Simple lowercase value - no quoting needed
	cap, err := NewTaggedUrnBuilder().Tag("key", "simple").Build()
	require.NoError(t, err)
	assert.Equal(t, "cap:key=simple", cap.ToString())

	// Value with spaces - needs quoting
	cap2, err := NewTaggedUrnBuilder().Tag("key", "has spaces").Build()
	require.NoError(t, err)
	assert.Equal(t, `cap:key="has spaces"`, cap2.ToString())

	// Value with semicolons - needs quoting
	cap3, err := NewTaggedUrnBuilder().Tag("key", "has;semi").Build()
	require.NoError(t, err)
	assert.Equal(t, `cap:key="has;semi"`, cap3.ToString())

	// Value with uppercase - needs quoting to preserve
	cap4, err := NewTaggedUrnBuilder().Tag("key", "HasUpper").Build()
	require.NoError(t, err)
	assert.Equal(t, `cap:key="HasUpper"`, cap4.ToString())

	// Value with quotes - needs quoting and escaping
	cap5, err := NewTaggedUrnBuilder().Tag("key", `has"quote`).Build()
	require.NoError(t, err)
	assert.Equal(t, `cap:key="has\"quote"`, cap5.ToString())

	// Value with backslashes - needs quoting and escaping
	cap6, err := NewTaggedUrnBuilder().Tag("key", `path\file`).Build()
	require.NoError(t, err)
	assert.Equal(t, `cap:key="path\\file"`, cap6.ToString())
}

func TestRoundTripSimple(t *testing.T) {
	original := "cap:op=generate;ext=pdf"
	cap, err := NewTaggedUrnFromString(original)
	require.NoError(t, err)
	serialized := cap.ToString()
	reparsed, err := NewTaggedUrnFromString(serialized)
	require.NoError(t, err)
	assert.True(t, cap.Equals(reparsed))
}

func TestRoundTripQuoted(t *testing.T) {
	original := `cap:key="Value With Spaces"`
	cap, err := NewTaggedUrnFromString(original)
	require.NoError(t, err)
	serialized := cap.ToString()
	reparsed, err := NewTaggedUrnFromString(serialized)
	require.NoError(t, err)
	assert.True(t, cap.Equals(reparsed))
	value, exists := reparsed.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "Value With Spaces", value)
}

func TestRoundTripEscapes(t *testing.T) {
	original := `cap:key="value\"with\\escapes"`
	cap, err := NewTaggedUrnFromString(original)
	require.NoError(t, err)
	value, exists := cap.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, `value"with\escapes`, value)
	serialized := cap.ToString()
	reparsed, err := NewTaggedUrnFromString(serialized)
	require.NoError(t, err)
	assert.True(t, cap.Equals(reparsed))
}

func TestMatchingCaseSensitiveValues(t *testing.T) {
	// Values with different case should NOT match
	cap1, err := NewTaggedUrnFromString(`cap:key="Value"`)
	require.NoError(t, err)
	cap2, err := NewTaggedUrnFromString(`cap:key="value"`)
	require.NoError(t, err)
	assert.False(t, cap1.Matches(cap2))
	assert.False(t, cap2.Matches(cap1))

	// Same case should match
	cap3, err := NewTaggedUrnFromString(`cap:key="Value"`)
	require.NoError(t, err)
	assert.True(t, cap1.Matches(cap3))
}

func TestBuilderPreservesCase(t *testing.T) {
	cap, err := NewTaggedUrnBuilder().
		Tag("KEY", "ValueWithCase").
		Build()
	require.NoError(t, err)

	// Key is lowercase
	value, exists := cap.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "ValueWithCase", value)

	// Value case preserved, so needs quoting
	assert.Equal(t, `cap:key="ValueWithCase"`, cap.ToString())
}

func TestHasTagCaseSensitive(t *testing.T) {
	cap, err := NewTaggedUrnFromString(`cap:key="Value"`)
	require.NoError(t, err)

	// Exact case match works
	assert.True(t, cap.HasTag("key", "Value"))

	// Different case does not match
	assert.False(t, cap.HasTag("key", "value"))
	assert.False(t, cap.HasTag("key", "VALUE"))

	// Key lookup is case-insensitive
	assert.True(t, cap.HasTag("KEY", "Value"))
	assert.True(t, cap.HasTag("Key", "Value"))
}

func TestWithTagPreservesValue(t *testing.T) {
	cap := NewTaggedUrnFromTags(map[string]string{})
	modified := cap.WithTag("key", "ValueWithCase")

	value, exists := modified.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "ValueWithCase", value)
}

func TestSemanticEquivalence(t *testing.T) {
	// Unquoted and quoted simple lowercase values are equivalent
	unquoted, err := NewTaggedUrnFromString("cap:key=simple")
	require.NoError(t, err)
	quoted, err := NewTaggedUrnFromString(`cap:key="simple"`)
	require.NoError(t, err)
	assert.True(t, unquoted.Equals(quoted))

	// Both serialize the same way (unquoted)
	assert.Equal(t, "cap:key=simple", unquoted.ToString())
	assert.Equal(t, "cap:key=simple", quoted.ToString())
}

func TestEmptyTaggedUrn(t *testing.T) {
	// Empty tagged URN should be valid and match everything
	empty, err := NewTaggedUrnFromString("cap:")
	assert.NoError(t, err)
	assert.NotNil(t, empty)
	assert.Equal(t, 0, len(empty.tags))
	assert.Equal(t, "cap:", empty.ToString())

	// Should match any other cap
	specific, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	assert.NoError(t, err)
	assert.True(t, empty.Matches(specific))
	assert.True(t, empty.Matches(empty))

	// With trailing semicolon
	empty2, err := NewTaggedUrnFromString("cap:;")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(empty2.tags))
}

func TestExtendedCharacterSupport(t *testing.T) {
	// Test forward slashes and colons in tag components
	cap, err := NewTaggedUrnFromString("cap:url=https://example_org/api;path=/some/file")
	assert.NoError(t, err)
	assert.NotNil(t, cap)

	url, exists := cap.GetTag("url")
	assert.True(t, exists)
	assert.Equal(t, "https://example_org/api", url)

	path, exists := cap.GetTag("path")
	assert.True(t, exists)
	assert.Equal(t, "/some/file", path)
}

func TestWildcardRestrictions(t *testing.T) {
	// Wildcard should be rejected in keys
	invalidKey, err := NewTaggedUrnFromString("cap:*=value")
	assert.Error(t, err)
	assert.Nil(t, invalidKey)
	capError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorInvalidCharacter, capError.Code)

	// Wildcard should be accepted in values
	validValue, err := NewTaggedUrnFromString("cap:key=*")
	assert.NoError(t, err)
	assert.NotNil(t, validValue)

	value, exists := validValue.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "*", value)
}

func TestDuplicateKeyRejection(t *testing.T) {
	// Duplicate keys should be rejected
	duplicate, err := NewTaggedUrnFromString("cap:key=value1;key=value2")
	assert.Error(t, err)
	assert.Nil(t, duplicate)
	capError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorDuplicateKey, capError.Code)
}

func TestNumericKeyRestriction(t *testing.T) {
	// Pure numeric keys should be rejected
	numericKey, err := NewTaggedUrnFromString("cap:123=value")
	assert.Error(t, err)
	assert.Nil(t, numericKey)
	capError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorNumericKey, capError.Code)

	// Mixed alphanumeric keys should be allowed
	mixedKey1, err := NewTaggedUrnFromString("cap:key123=value")
	assert.NoError(t, err)
	assert.NotNil(t, mixedKey1)

	mixedKey2, err := NewTaggedUrnFromString("cap:123key=value")
	assert.NoError(t, err)
	assert.NotNil(t, mixedKey2)

	// Pure numeric values should be allowed
	numericValue, err := NewTaggedUrnFromString("cap:key=123")
	assert.NoError(t, err)
	assert.NotNil(t, numericValue)

	value, exists := numericValue.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "123", value)
}

func TestEmptyValueError(t *testing.T) {
	cap, err := NewTaggedUrnFromString("cap:key=")
	assert.Nil(t, cap)
	assert.Error(t, err)

	cap2, err := NewTaggedUrnFromString("cap:key=;other=value")
	assert.Nil(t, cap2)
	assert.Error(t, err)
}

// ============================================================================
// MATCHING SEMANTICS SPECIFICATION TESTS
// These 9 tests verify the exact matching semantics from RULES.md Sections 12-17
// All implementations (Rust, Go, JS, ObjC) must pass these identically
// ============================================================================

func TestMatchingSemantics_Test1_ExactMatch(t *testing.T) {
	// Test 1: Exact match
	// Cap:     cap:op=generate;ext=pdf
	// Request: cap:op=generate;ext=pdf
	// Result:  MATCH
	cap, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	assert.True(t, cap.Matches(request), "Test 1: Exact match should succeed")
}

func TestMatchingSemantics_Test2_CapMissingTag(t *testing.T) {
	// Test 2: Cap missing tag (implicit wildcard)
	// Cap:     cap:op=generate
	// Request: cap:op=generate;ext=pdf
	// Result:  MATCH (cap can handle any ext)
	cap, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	assert.True(t, cap.Matches(request), "Test 2: Cap missing tag should match (implicit wildcard)")
}

func TestMatchingSemantics_Test3_CapHasExtraTag(t *testing.T) {
	// Test 3: Cap has extra tag
	// Cap:     cap:op=generate;ext=pdf;version=2
	// Request: cap:op=generate;ext=pdf
	// Result:  MATCH (request doesn't constrain version)
	cap, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;version=2")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	assert.True(t, cap.Matches(request), "Test 3: Cap with extra tag should match")
}

func TestMatchingSemantics_Test4_RequestHasWildcard(t *testing.T) {
	// Test 4: Request has wildcard
	// Cap:     cap:op=generate;ext=pdf
	// Request: cap:op=generate;ext=*
	// Result:  MATCH (request accepts any ext)
	cap, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=*")
	require.NoError(t, err)

	assert.True(t, cap.Matches(request), "Test 4: Request wildcard should match")
}

func TestMatchingSemantics_Test5_CapHasWildcard(t *testing.T) {
	// Test 5: Cap has wildcard
	// Cap:     cap:op=generate;ext=*
	// Request: cap:op=generate;ext=pdf
	// Result:  MATCH (cap handles any ext)
	cap, err := NewTaggedUrnFromString("cap:op=generate;ext=*")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	assert.True(t, cap.Matches(request), "Test 5: Cap wildcard should match")
}

func TestMatchingSemantics_Test6_ValueMismatch(t *testing.T) {
	// Test 6: Value mismatch
	// Cap:     cap:op=generate;ext=pdf
	// Request: cap:op=generate;ext=docx
	// Result:  NO MATCH
	cap, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=docx")
	require.NoError(t, err)

	assert.False(t, cap.Matches(request), "Test 6: Value mismatch should not match")
}

func TestMatchingSemantics_Test7_FallbackPattern(t *testing.T) {
	// Test 7: Fallback pattern
	// Cap:     cap:op=generate_thumbnail;out=std:binary.v1
	// Request: cap:op=generate_thumbnail;out=std:binary.v1;ext=wav
	// Result:  MATCH (cap has implicit ext=*)
	cap, err := NewTaggedUrnFromString("cap:op=generate_thumbnail;out=std:binary.v1")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate_thumbnail;out=std:binary.v1;ext=wav")
	require.NoError(t, err)

	assert.True(t, cap.Matches(request), "Test 7: Fallback pattern should match (cap missing ext = implicit wildcard)")
}

func TestMatchingSemantics_Test8_EmptyCapMatchesAnything(t *testing.T) {
	// Test 8: Empty cap matches anything
	// Cap:     cap:
	// Request: cap:op=generate;ext=pdf
	// Result:  MATCH
	cap, err := NewTaggedUrnFromString("cap:")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	assert.True(t, cap.Matches(request), "Test 8: Empty cap should match anything")
}

func TestMatchingSemantics_Test9_CrossDimensionIndependence(t *testing.T) {
	// Test 9: Cross-dimension independence
	// Cap:     cap:op=generate
	// Request: cap:ext=pdf
	// Result:  MATCH (both have implicit wildcards for missing tags)
	cap, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:ext=pdf")
	require.NoError(t, err)

	assert.True(t, cap.Matches(request), "Test 9: Cross-dimension independence should match")
}
