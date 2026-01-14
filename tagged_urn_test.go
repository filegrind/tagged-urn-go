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
	assert.Equal(t, "cap", taggedUrn.GetPrefix())

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

func TestCustomPrefix(t *testing.T) {
	urn, err := NewTaggedUrnFromString("myapp:op=generate;ext=pdf")
	require.NoError(t, err)

	assert.Equal(t, "myapp", urn.GetPrefix())
	op, exists := urn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)
	assert.Equal(t, "myapp:ext=pdf;op=generate", urn.ToString())
}

func TestPrefixCaseInsensitive(t *testing.T) {
	urn1, err := NewTaggedUrnFromString("CAP:op=test")
	require.NoError(t, err)
	urn2, err := NewTaggedUrnFromString("cap:op=test")
	require.NoError(t, err)
	urn3, err := NewTaggedUrnFromString("Cap:op=test")
	require.NoError(t, err)

	assert.Equal(t, "cap", urn1.GetPrefix())
	assert.Equal(t, "cap", urn2.GetPrefix())
	assert.Equal(t, "cap", urn3.GetPrefix())
	assert.True(t, urn1.Equals(urn2))
	assert.True(t, urn2.Equals(urn3))
}

func TestPrefixMismatchError(t *testing.T) {
	urn1, err := NewTaggedUrnFromString("cap:op=test")
	require.NoError(t, err)
	urn2, err := NewTaggedUrnFromString("myapp:op=test")
	require.NoError(t, err)

	_, err = urn1.Matches(urn2)
	assert.Error(t, err)
	capError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorPrefixMismatch, capError.Code)
}

func TestBuilderWithPrefix(t *testing.T) {
	urn, err := NewTaggedUrnBuilder("custom").
		Tag("key", "value").
		Build()
	require.NoError(t, err)

	assert.Equal(t, "custom", urn.GetPrefix())
	assert.Equal(t, "custom:key=value", urn.ToString())
}

func TestCanonicalStringFormat(t *testing.T) {
	taggedUrn, err := NewTaggedUrnFromString("cap:op=generate;target=thumbnail;ext=pdf")
	require.NoError(t, err)

	// Should be sorted alphabetically and have no trailing semicolon in canonical form
	assert.Equal(t, "cap:ext=pdf;op=generate;target=thumbnail", taggedUrn.ToString())
}

func TestPrefixRequired(t *testing.T) {
	// Missing prefix should fail
	taggedUrn, err := NewTaggedUrnFromString("op=generate;ext=pdf")
	assert.Nil(t, taggedUrn)
	assert.Error(t, err)
	assert.Equal(t, ErrorMissingPrefix, err.(*TaggedUrnError).Code)

	// Empty prefix should fail
	taggedUrn, err = NewTaggedUrnFromString(":op=generate")
	assert.Nil(t, taggedUrn)
	assert.Error(t, err)
	assert.Equal(t, ErrorEmptyPrefix, err.(*TaggedUrnError).Code)

	// Valid prefix should work
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
	urn1, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	urn2, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;")
	require.NoError(t, err)

	// They should be equal
	assert.True(t, urn1.Equals(urn2))

	// They should have same hash
	assert.Equal(t, urn1.Hash(), urn2.Hash())

	// They should have same string representation (canonical form)
	assert.Equal(t, urn1.ToString(), urn2.ToString())

	// They should match each other
	matches1, err := urn1.Matches(urn2)
	require.NoError(t, err)
	assert.True(t, matches1)

	matches2, err := urn2.Matches(urn1)
	require.NoError(t, err)
	assert.True(t, matches2)
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
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;target=thumbnail")
	require.NoError(t, err)

	// Exact match
	request1, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;target=thumbnail")
	require.NoError(t, err)
	matches, err := urn.Matches(request1)
	require.NoError(t, err)
	assert.True(t, matches)

	// Subset match
	request2, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)
	matches, err = urn.Matches(request2)
	require.NoError(t, err)
	assert.True(t, matches)

	// Wildcard match
	request3, err := NewTaggedUrnFromString("cap:ext=*")
	require.NoError(t, err)
	matches, err = urn.Matches(request3)
	require.NoError(t, err)
	assert.True(t, matches)

	// No match - conflicting value
	request4, err := NewTaggedUrnFromString("cap:op=extract")
	require.NoError(t, err)
	matches, err = urn.Matches(request4)
	require.NoError(t, err)
	assert.False(t, matches)
}

func TestMissingTagHandling(t *testing.T) {
	urn, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	// Request with missing tag should match (URN missing format tag = wildcard, can handle any format)
	request1, err := NewTaggedUrnFromString("cap:ext=pdf")
	require.NoError(t, err)
	matches, err := urn.Matches(request1)
	require.NoError(t, err)
	assert.True(t, matches)

	// But URN with extra tags can match subset requests
	urn2, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)
	request2, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)
	matches, err = urn2.Matches(request2)
	require.NoError(t, err)
	assert.True(t, matches)
}

func TestSpecificity(t *testing.T) {
	urn1, err := NewTaggedUrnFromString("cap:op=*")
	require.NoError(t, err)

	urn2, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	urn3, err := NewTaggedUrnFromString("cap:op=*;ext=pdf")
	require.NoError(t, err)

	assert.Equal(t, 0, urn1.Specificity()) // wildcard doesn't count
	assert.Equal(t, 1, urn2.Specificity())
	assert.Equal(t, 1, urn3.Specificity()) // only ext=pdf counts, op=* doesn't count

	moreSpecific, err := urn2.IsMoreSpecificThan(urn1)
	require.NoError(t, err)
	assert.True(t, moreSpecific)
}

func TestCompatibility(t *testing.T) {
	urn1, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	urn2, err := NewTaggedUrnFromString("cap:op=generate;format=*")
	require.NoError(t, err)

	urn3, err := NewTaggedUrnFromString("cap:op=extract;ext=pdf")
	require.NoError(t, err)

	compatible, err := urn1.IsCompatibleWith(urn2)
	require.NoError(t, err)
	assert.True(t, compatible)

	compatible, err = urn2.IsCompatibleWith(urn1)
	require.NoError(t, err)
	assert.True(t, compatible)

	compatible, err = urn1.IsCompatibleWith(urn3)
	require.NoError(t, err)
	assert.False(t, compatible)

	// Missing tags are treated as wildcards for compatibility
	urn4, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	compatible, err = urn1.IsCompatibleWith(urn4)
	require.NoError(t, err)
	assert.True(t, compatible)

	compatible, err = urn4.IsCompatibleWith(urn1)
	require.NoError(t, err)
	assert.True(t, compatible)
}

func TestConvenienceMethods(t *testing.T) {
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;output=binary;target=thumbnail")
	require.NoError(t, err)

	op, exists := urn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)

	target, exists := urn.GetTag("target")
	assert.True(t, exists)
	assert.Equal(t, "thumbnail", target)

	format, exists := urn.GetTag("ext")
	assert.True(t, exists)
	assert.Equal(t, "pdf", format)

	output, exists := urn.GetTag("output")
	assert.True(t, exists)
	assert.Equal(t, "binary", output)
}

func TestBuilder(t *testing.T) {
	urn, err := NewTaggedUrnBuilder("cap").
		Tag("op", "generate").
		Tag("target", "thumbnail").
		Tag("ext", "pdf").
		Tag("output", "binary").
		Build()
	require.NoError(t, err)

	op, exists := urn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)

	output, exists := urn.GetTag("output")
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
	urn, err := NewTaggedUrnFromString("cap:ext=pdf")
	require.NoError(t, err)

	wildcarded := urn.WithWildcardTag("ext")

	assert.Equal(t, "cap:ext=*", wildcarded.ToString())

	// Test that wildcarded URN can match more requests
	request, err := NewTaggedUrnFromString("cap:ext=jpg")
	require.NoError(t, err)
	matches, err := urn.Matches(request)
	require.NoError(t, err)
	assert.False(t, matches)

	wildcardRequest, err := NewTaggedUrnFromString("cap:ext=*")
	require.NoError(t, err)
	matches, err = wildcarded.Matches(wildcardRequest)
	require.NoError(t, err)
	assert.True(t, matches)
}

func TestSubset(t *testing.T) {
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;output=binary;target=thumbnail;")
	require.NoError(t, err)

	subset := urn.Subset([]string{"type", "ext"})

	assert.Equal(t, "cap:ext=pdf", subset.ToString())
}

func TestMerge(t *testing.T) {
	urn1, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	urn2, err := NewTaggedUrnFromString("cap:ext=pdf;output=binary")
	require.NoError(t, err)

	merged, err := urn1.Merge(urn2)
	require.NoError(t, err)

	assert.Equal(t, "cap:ext=pdf;op=generate;output=binary", merged.ToString())
}

func TestMergePrefixMismatch(t *testing.T) {
	urn1, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	urn2, err := NewTaggedUrnFromString("myapp:ext=pdf")
	require.NoError(t, err)

	_, err = urn1.Merge(urn2)
	assert.Error(t, err)
	capError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorPrefixMismatch, capError.Code)
}

func TestEquality(t *testing.T) {
	urn1, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	urn2, err := NewTaggedUrnFromString("cap:op=generate") // different order
	require.NoError(t, err)

	urn3, err := NewTaggedUrnFromString("cap:op=generate;type=image")
	require.NoError(t, err)

	assert.True(t, urn1.Equals(urn2)) // order doesn't matter
	assert.False(t, urn1.Equals(urn3))
}

func TestEqualityDifferentPrefix(t *testing.T) {
	urn1, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	urn2, err := NewTaggedUrnFromString("myapp:op=generate")
	require.NoError(t, err)

	assert.False(t, urn1.Equals(urn2))
}

func TestCapMatcher(t *testing.T) {
	matcher := &CapMatcher{}

	urns := []*TaggedUrn{}

	urn1, err := NewTaggedUrnFromString("cap:op=*")
	require.NoError(t, err)
	urns = append(urns, urn1)

	urn2, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)
	urns = append(urns, urn2)

	urn3, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)
	urns = append(urns, urn3)

	request, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	best, err := matcher.FindBestMatch(urns, request)
	require.NoError(t, err)
	require.NotNil(t, best)

	// Most specific URN that can handle the request
	assert.Equal(t, "cap:ext=pdf;op=generate", best.ToString())
}

func TestCapMatcherPrefixMismatch(t *testing.T) {
	matcher := &CapMatcher{}

	urns := []*TaggedUrn{}

	urn1, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)
	urns = append(urns, urn1)

	request, err := NewTaggedUrnFromString("myapp:op=generate")
	require.NoError(t, err)

	_, err = matcher.FindBestMatch(urns, request)
	assert.Error(t, err)
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

func TestJSONSerializationWithCustomPrefix(t *testing.T) {
	original, err := NewTaggedUrnFromString("myapp:key=value")
	require.NoError(t, err)

	data, err := json.Marshal(original)
	assert.NoError(t, err)

	var decoded TaggedUrn
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.True(t, original.Equals(&decoded))
	assert.Equal(t, "myapp", decoded.GetPrefix())
}

func TestUnquotedValuesLowercased(t *testing.T) {
	// Unquoted values are normalized to lowercase
	urn, err := NewTaggedUrnFromString("cap:OP=Generate;EXT=PDF;Target=Thumbnail;")
	require.NoError(t, err)

	// Keys are always lowercase
	op, exists := urn.GetTag("op")
	assert.True(t, exists)
	assert.Equal(t, "generate", op)

	ext, exists := urn.GetTag("ext")
	assert.True(t, exists)
	assert.Equal(t, "pdf", ext)

	target, exists := urn.GetTag("target")
	assert.True(t, exists)
	assert.Equal(t, "thumbnail", target)

	// Key lookup is case-insensitive
	op2, exists := urn.GetTag("OP")
	assert.True(t, exists)
	assert.Equal(t, "generate", op2)

	// Both URNs parse to same lowercase values
	urn2, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;target=thumbnail;")
	require.NoError(t, err)
	assert.Equal(t, urn.ToString(), urn2.ToString())
	assert.True(t, urn.Equals(urn2))
}

func TestQuotedValuesPreserveCase(t *testing.T) {
	// Quoted values preserve their case
	urn, err := NewTaggedUrnFromString(`cap:key="Value With Spaces"`)
	require.NoError(t, err)
	value, exists := urn.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "Value With Spaces", value)

	// Key is still lowercase
	urn2, err := NewTaggedUrnFromString(`cap:KEY="Value With Spaces"`)
	require.NoError(t, err)
	value2, exists := urn2.GetTag("key")
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
	urn, err := NewTaggedUrnFromString(`cap:key="value;with;semicolons"`)
	require.NoError(t, err)
	value, exists := urn.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "value;with;semicolons", value)

	// Equals in quoted values
	urn2, err := NewTaggedUrnFromString(`cap:key="value=with=equals"`)
	require.NoError(t, err)
	value2, exists := urn2.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "value=with=equals", value2)

	// Spaces in quoted values
	urn3, err := NewTaggedUrnFromString(`cap:key="hello world"`)
	require.NoError(t, err)
	value3, exists := urn3.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "hello world", value3)
}

func TestQuotedValueEscapeSequences(t *testing.T) {
	// Escaped quotes
	urn, err := NewTaggedUrnFromString(`cap:key="value\"quoted\""`)
	require.NoError(t, err)
	value, exists := urn.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, `value"quoted"`, value)

	// Escaped backslashes
	urn2, err := NewTaggedUrnFromString(`cap:key="path\\file"`)
	require.NoError(t, err)
	value2, exists := urn2.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, `path\file`, value2)

	// Mixed escapes
	urn3, err := NewTaggedUrnFromString(`cap:key="say \"hello\\world\""`)
	require.NoError(t, err)
	value3, exists := urn3.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, `say "hello\world"`, value3)
}

func TestMixedQuotedUnquoted(t *testing.T) {
	urn, err := NewTaggedUrnFromString(`cap:a="Quoted";b=simple`)
	require.NoError(t, err)

	a, exists := urn.GetTag("a")
	assert.True(t, exists)
	assert.Equal(t, "Quoted", a)

	b, exists := urn.GetTag("b")
	assert.True(t, exists)
	assert.Equal(t, "simple", b)
}

func TestUnterminatedQuoteError(t *testing.T) {
	urn, err := NewTaggedUrnFromString(`cap:key="unterminated`)
	assert.Nil(t, urn)
	assert.Error(t, err)
	urnError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorUnterminatedQuote, urnError.Code)
}

func TestInvalidEscapeSequenceError(t *testing.T) {
	urn, err := NewTaggedUrnFromString(`cap:key="bad\n"`)
	assert.Nil(t, urn)
	assert.Error(t, err)
	urnError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorInvalidEscapeSequence, urnError.Code)

	// Invalid escape at end
	urn2, err := NewTaggedUrnFromString(`cap:key="bad\x"`)
	assert.Nil(t, urn2)
	assert.Error(t, err)
	urnError2, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorInvalidEscapeSequence, urnError2.Code)
}

func TestSerializationSmartQuoting(t *testing.T) {
	// Simple lowercase value - no quoting needed
	urn, err := NewTaggedUrnBuilder("cap").Tag("key", "simple").Build()
	require.NoError(t, err)
	assert.Equal(t, "cap:key=simple", urn.ToString())

	// Value with spaces - needs quoting
	urn2, err := NewTaggedUrnBuilder("cap").Tag("key", "has spaces").Build()
	require.NoError(t, err)
	assert.Equal(t, `cap:key="has spaces"`, urn2.ToString())

	// Value with semicolons - needs quoting
	urn3, err := NewTaggedUrnBuilder("cap").Tag("key", "has;semi").Build()
	require.NoError(t, err)
	assert.Equal(t, `cap:key="has;semi"`, urn3.ToString())

	// Value with uppercase - needs quoting to preserve
	urn4, err := NewTaggedUrnBuilder("cap").Tag("key", "HasUpper").Build()
	require.NoError(t, err)
	assert.Equal(t, `cap:key="HasUpper"`, urn4.ToString())

	// Value with quotes - needs quoting and escaping
	urn5, err := NewTaggedUrnBuilder("cap").Tag("key", `has"quote`).Build()
	require.NoError(t, err)
	assert.Equal(t, `cap:key="has\"quote"`, urn5.ToString())

	// Value with backslashes - needs quoting and escaping
	urn6, err := NewTaggedUrnBuilder("cap").Tag("key", `path\file`).Build()
	require.NoError(t, err)
	assert.Equal(t, `cap:key="path\\file"`, urn6.ToString())
}

func TestRoundTripSimple(t *testing.T) {
	original := "cap:op=generate;ext=pdf"
	urn, err := NewTaggedUrnFromString(original)
	require.NoError(t, err)
	serialized := urn.ToString()
	reparsed, err := NewTaggedUrnFromString(serialized)
	require.NoError(t, err)
	assert.True(t, urn.Equals(reparsed))
}

func TestRoundTripQuoted(t *testing.T) {
	original := `cap:key="Value With Spaces"`
	urn, err := NewTaggedUrnFromString(original)
	require.NoError(t, err)
	serialized := urn.ToString()
	reparsed, err := NewTaggedUrnFromString(serialized)
	require.NoError(t, err)
	assert.True(t, urn.Equals(reparsed))
	value, exists := reparsed.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "Value With Spaces", value)
}

func TestRoundTripEscapes(t *testing.T) {
	original := `cap:key="value\"with\\escapes"`
	urn, err := NewTaggedUrnFromString(original)
	require.NoError(t, err)
	value, exists := urn.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, `value"with\escapes`, value)
	serialized := urn.ToString()
	reparsed, err := NewTaggedUrnFromString(serialized)
	require.NoError(t, err)
	assert.True(t, urn.Equals(reparsed))
}

func TestMatchingCaseSensitiveValues(t *testing.T) {
	// Values with different case should NOT match
	urn1, err := NewTaggedUrnFromString(`cap:key="Value"`)
	require.NoError(t, err)
	urn2, err := NewTaggedUrnFromString(`cap:key="value"`)
	require.NoError(t, err)

	matches1, err := urn1.Matches(urn2)
	require.NoError(t, err)
	assert.False(t, matches1)

	matches2, err := urn2.Matches(urn1)
	require.NoError(t, err)
	assert.False(t, matches2)

	// Same case should match
	urn3, err := NewTaggedUrnFromString(`cap:key="Value"`)
	require.NoError(t, err)
	matches3, err := urn1.Matches(urn3)
	require.NoError(t, err)
	assert.True(t, matches3)
}

func TestBuilderPreservesCase(t *testing.T) {
	urn, err := NewTaggedUrnBuilder("cap").
		Tag("KEY", "ValueWithCase").
		Build()
	require.NoError(t, err)

	// Key is lowercase
	value, exists := urn.GetTag("key")
	assert.True(t, exists)
	assert.Equal(t, "ValueWithCase", value)

	// Value case preserved, so needs quoting
	assert.Equal(t, `cap:key="ValueWithCase"`, urn.ToString())
}

func TestHasTagCaseSensitive(t *testing.T) {
	urn, err := NewTaggedUrnFromString(`cap:key="Value"`)
	require.NoError(t, err)

	// Exact case match works
	assert.True(t, urn.HasTag("key", "Value"))

	// Different case does not match
	assert.False(t, urn.HasTag("key", "value"))
	assert.False(t, urn.HasTag("key", "VALUE"))

	// Key lookup is case-insensitive
	assert.True(t, urn.HasTag("KEY", "Value"))
	assert.True(t, urn.HasTag("Key", "Value"))
}

func TestWithTagPreservesValue(t *testing.T) {
	urn := NewTaggedUrnFromTags("cap", map[string]string{})
	modified := urn.WithTag("key", "ValueWithCase")

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
	assert.Equal(t, "cap:", empty.ToString())

	// Should match any other URN with same prefix
	specific, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	assert.NoError(t, err)

	matches, err := empty.Matches(specific)
	require.NoError(t, err)
	assert.True(t, matches)

	matches, err = empty.Matches(empty)
	require.NoError(t, err)
	assert.True(t, matches)

	// With trailing semicolon
	empty2, err := NewTaggedUrnFromString("cap:;")
	assert.NoError(t, err)
	assert.Equal(t, "cap:", empty2.ToString())
}

func TestEmptyWithCustomPrefix(t *testing.T) {
	empty, err := NewTaggedUrnFromString("myapp:")
	require.NoError(t, err)
	assert.Equal(t, "myapp", empty.GetPrefix())
	assert.Equal(t, "myapp:", empty.ToString())
}

func TestExtendedCharacterSupport(t *testing.T) {
	// Test forward slashes and colons in tag components
	urn, err := NewTaggedUrnFromString("cap:url=https://example_org/api;path=/some/file")
	assert.NoError(t, err)
	assert.NotNil(t, urn)

	url, exists := urn.GetTag("url")
	assert.True(t, exists)
	assert.Equal(t, "https://example_org/api", url)

	path, exists := urn.GetTag("path")
	assert.True(t, exists)
	assert.Equal(t, "/some/file", path)
}

func TestWildcardRestrictions(t *testing.T) {
	// Wildcard should be rejected in keys
	invalidKey, err := NewTaggedUrnFromString("cap:*=value")
	assert.Error(t, err)
	assert.Nil(t, invalidKey)
	urnError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorInvalidCharacter, urnError.Code)

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
	urnError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorDuplicateKey, urnError.Code)
}

func TestNumericKeyRestriction(t *testing.T) {
	// Pure numeric keys should be rejected
	numericKey, err := NewTaggedUrnFromString("cap:123=value")
	assert.Error(t, err)
	assert.Nil(t, numericKey)
	urnError, ok := err.(*TaggedUrnError)
	assert.True(t, ok)
	assert.Equal(t, ErrorNumericKey, urnError.Code)

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
	urn, err := NewTaggedUrnFromString("cap:key=")
	assert.Nil(t, urn)
	assert.Error(t, err)

	urn2, err := NewTaggedUrnFromString("cap:key=;other=value")
	assert.Nil(t, urn2)
	assert.Error(t, err)
}

func TestMatchingDifferentPrefixesError(t *testing.T) {
	// URNs with different prefixes should cause an error, not just return false
	urn1, err := NewTaggedUrnFromString("cap:op=test")
	require.NoError(t, err)
	urn2, err := NewTaggedUrnFromString("other:op=test")
	require.NoError(t, err)

	_, err = urn1.Matches(urn2)
	assert.Error(t, err)

	_, err = urn1.IsCompatibleWith(urn2)
	assert.Error(t, err)

	_, err = urn1.IsMoreSpecificThan(urn2)
	assert.Error(t, err)
}

// ============================================================================
// MATCHING SEMANTICS SPECIFICATION TESTS
// These 9 tests verify the exact matching semantics from RULES.md Sections 12-17
// All implementations (Rust, Go, JS, ObjC) must pass these identically
// ============================================================================

func TestMatchingSemantics_Test1_ExactMatch(t *testing.T) {
	// Test 1: Exact match
	// URN:     cap:op=generate;ext=pdf
	// Request: cap:op=generate;ext=pdf
	// Result:  MATCH
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	matches, err := urn.Matches(request)
	require.NoError(t, err)
	assert.True(t, matches, "Test 1: Exact match should succeed")
}

func TestMatchingSemantics_Test2_UrnMissingTag(t *testing.T) {
	// Test 2: URN missing tag (implicit wildcard)
	// URN:     cap:op=generate
	// Request: cap:op=generate;ext=pdf
	// Result:  MATCH (URN can handle any ext)
	urn, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	matches, err := urn.Matches(request)
	require.NoError(t, err)
	assert.True(t, matches, "Test 2: URN missing tag should match (implicit wildcard)")
}

func TestMatchingSemantics_Test3_UrnHasExtraTag(t *testing.T) {
	// Test 3: URN has extra tag
	// URN:     cap:op=generate;ext=pdf;version=2
	// Request: cap:op=generate;ext=pdf
	// Result:  MATCH (request doesn't constrain version)
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf;version=2")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	matches, err := urn.Matches(request)
	require.NoError(t, err)
	assert.True(t, matches, "Test 3: URN with extra tag should match")
}

func TestMatchingSemantics_Test4_RequestHasWildcard(t *testing.T) {
	// Test 4: Request has wildcard
	// URN:     cap:op=generate;ext=pdf
	// Request: cap:op=generate;ext=*
	// Result:  MATCH (request accepts any ext)
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=*")
	require.NoError(t, err)

	matches, err := urn.Matches(request)
	require.NoError(t, err)
	assert.True(t, matches, "Test 4: Request wildcard should match")
}

func TestMatchingSemantics_Test5_UrnHasWildcard(t *testing.T) {
	// Test 5: URN has wildcard
	// URN:     cap:op=generate;ext=*
	// Request: cap:op=generate;ext=pdf
	// Result:  MATCH (URN handles any ext)
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext=*")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	matches, err := urn.Matches(request)
	require.NoError(t, err)
	assert.True(t, matches, "Test 5: URN wildcard should match")
}

func TestMatchingSemantics_Test6_ValueMismatch(t *testing.T) {
	// Test 6: Value mismatch
	// URN:     cap:op=generate;ext=pdf
	// Request: cap:op=generate;ext=docx
	// Result:  NO MATCH
	urn, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=docx")
	require.NoError(t, err)

	matches, err := urn.Matches(request)
	require.NoError(t, err)
	assert.False(t, matches, "Test 6: Value mismatch should not match")
}

func TestMatchingSemantics_Test7_FallbackPattern(t *testing.T) {
	// Test 7: Fallback pattern
	// URN:     cap:op=generate_thumbnail;out=std:binary.v1
	// Request: cap:op=generate_thumbnail;out=std:binary.v1;ext=wav
	// Result:  MATCH (URN has implicit ext=*)
	urn, err := NewTaggedUrnFromString("cap:op=generate_thumbnail;out=std:binary.v1")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate_thumbnail;out=std:binary.v1;ext=wav")
	require.NoError(t, err)

	matches, err := urn.Matches(request)
	require.NoError(t, err)
	assert.True(t, matches, "Test 7: Fallback pattern should match (URN missing ext = implicit wildcard)")
}

func TestMatchingSemantics_Test8_EmptyUrnMatchesAnything(t *testing.T) {
	// Test 8: Empty URN matches anything
	// URN:     cap:
	// Request: cap:op=generate;ext=pdf
	// Result:  MATCH
	urn, err := NewTaggedUrnFromString("cap:")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:op=generate;ext=pdf")
	require.NoError(t, err)

	matches, err := urn.Matches(request)
	require.NoError(t, err)
	assert.True(t, matches, "Test 8: Empty URN should match anything")
}

func TestMatchingSemantics_Test9_CrossDimensionIndependence(t *testing.T) {
	// Test 9: Cross-dimension independence
	// URN:     cap:op=generate
	// Request: cap:ext=pdf
	// Result:  MATCH (both have implicit wildcards for missing tags)
	urn, err := NewTaggedUrnFromString("cap:op=generate")
	require.NoError(t, err)

	request, err := NewTaggedUrnFromString("cap:ext=pdf")
	require.NoError(t, err)

	matches, err := urn.Matches(request)
	require.NoError(t, err)
	assert.True(t, matches, "Test 9: Cross-dimension independence should match")
}
