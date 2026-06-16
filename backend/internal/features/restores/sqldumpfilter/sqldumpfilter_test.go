package sqldumpfilter

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildDump returns a minimal mysqldump-style SQL stream with two tables: users and orders.
// The structure mirrors real mysqldump output: a preamble, two table sections (DDL + data each),
// and a trailing comment after the last UNLOCK TABLES.
func buildDump() string {
	return strings.Join([]string{
		"SET NAMES utf8mb4;",
		"",
		"-- Table structure for table `users`",
		"",
		"CREATE TABLE `users` (id INT);",
		"LOCK TABLES `users` WRITE;",
		"",
		"-- Dumping data for table `users`",
		"",
		"INSERT INTO `users` VALUES (1),(2);",
		"",
		"UNLOCK TABLES;",
		"",
		"-- Table structure for table `orders`",
		"",
		"CREATE TABLE `orders` (total INT);",
		"LOCK TABLES `orders` WRITE;",
		"",
		"-- Dumping data for table `orders`",
		"",
		"INSERT INTO `orders` VALUES (100);",
		"",
		"UNLOCK TABLES;",
		"",
		"-- Dump completed on 2026-01-01",
	}, "\n") + "\n"
}

func applyFilter(t *testing.T, dump string, include, exclude []string) string {
	t.Helper()

	output, err := io.ReadAll(NewTableFilterReader(strings.NewReader(dump), include, exclude))
	require.NoError(t, err)

	return string(output)
}

func Test_NewTableFilterReader_WithNoFilter_ReturnsDumpUnchanged(t *testing.T) {
	dump := buildDump()

	// When both slices are empty the original reader is returned directly (no pipe),
	// so the output must be byte-for-byte identical to the input.
	output := applyFilter(t, dump, nil, nil)

	assert.Equal(t, dump, output)
}

func Test_NewTableFilterReader_WithIncludeFilter(t *testing.T) {
	dump := buildDump()

	t.Run("Include users: emits users section, drops orders section", func(t *testing.T) {
		output := applyFilter(t, dump, []string{"users"}, nil)

		assert.Contains(t, output, "CREATE TABLE `users`")
		assert.Contains(t, output, "INSERT INTO `users` VALUES")
		assert.NotContains(t, output, "CREATE TABLE `orders`")
		assert.NotContains(t, output, "INSERT INTO `orders` VALUES")
	})

	t.Run("Include orders: emits orders section, drops users section", func(t *testing.T) {
		output := applyFilter(t, dump, []string{"orders"}, nil)

		assert.Contains(t, output, "CREATE TABLE `orders`")
		assert.Contains(t, output, "INSERT INTO `orders` VALUES")
		assert.NotContains(t, output, "CREATE TABLE `users`")
		assert.NotContains(t, output, "INSERT INTO `users` VALUES")
	})

	t.Run("Include both tables: emits all table sections", func(t *testing.T) {
		output := applyFilter(t, dump, []string{"users", "orders"}, nil)

		assert.Contains(t, output, "CREATE TABLE `users`")
		assert.Contains(t, output, "CREATE TABLE `orders`")
		assert.Contains(t, output, "INSERT INTO `users` VALUES")
		assert.Contains(t, output, "INSERT INTO `orders` VALUES")
	})

	t.Run("Include non-existent table: emits only preamble and epilogue", func(t *testing.T) {
		output := applyFilter(t, dump, []string{"products"}, nil)

		assert.Contains(t, output, "SET NAMES utf8mb4;")
		assert.Contains(t, output, "-- Dump completed")
		assert.NotContains(t, output, "CREATE TABLE")
		assert.NotContains(t, output, "INSERT INTO")
	})

	t.Run("Section header lines are included only for the included table", func(t *testing.T) {
		output := applyFilter(t, dump, []string{"users"}, nil)

		assert.Contains(t, output, "-- Table structure for table `users`")
		assert.Contains(t, output, "-- Dumping data for table `users`")
		assert.NotContains(t, output, "-- Table structure for table `orders`")
		assert.NotContains(t, output, "-- Dumping data for table `orders`")
	})

	t.Run("UNLOCK TABLES is emitted exactly once for a single included table", func(t *testing.T) {
		output := applyFilter(t, dump, []string{"users"}, nil)

		assert.Equal(t, 1, strings.Count(output, "UNLOCK TABLES;"))
	})

	t.Run("Preamble (before the first table section) is always emitted", func(t *testing.T) {
		output := applyFilter(t, dump, []string{"users"}, nil)

		assert.Contains(t, output, "SET NAMES utf8mb4;")
	})

	t.Run("Epilogue (after the last UNLOCK TABLES) is always emitted", func(t *testing.T) {
		output := applyFilter(t, dump, []string{"users"}, nil)

		assert.Contains(t, output, "-- Dump completed on 2026-01-01")
	})
}

func Test_NewTableFilterReader_WithExcludeFilter(t *testing.T) {
	dump := buildDump()

	t.Run("Exclude users: emits orders section, drops users section", func(t *testing.T) {
		output := applyFilter(t, dump, nil, []string{"users"})

		assert.Contains(t, output, "CREATE TABLE `orders`")
		assert.Contains(t, output, "INSERT INTO `orders` VALUES")
		assert.NotContains(t, output, "CREATE TABLE `users`")
		assert.NotContains(t, output, "INSERT INTO `users` VALUES")
	})

	t.Run("Exclude orders: emits users section, drops orders section", func(t *testing.T) {
		output := applyFilter(t, dump, nil, []string{"orders"})

		assert.Contains(t, output, "CREATE TABLE `users`")
		assert.Contains(t, output, "INSERT INTO `users` VALUES")
		assert.NotContains(t, output, "CREATE TABLE `orders`")
		assert.NotContains(t, output, "INSERT INTO `orders` VALUES")
	})

	t.Run("Exclude all tables: emits only preamble and epilogue", func(t *testing.T) {
		output := applyFilter(t, dump, nil, []string{"users", "orders"})

		assert.Contains(t, output, "SET NAMES utf8mb4;")
		assert.Contains(t, output, "-- Dump completed")
		assert.NotContains(t, output, "CREATE TABLE")
		assert.NotContains(t, output, "INSERT INTO")
	})

	t.Run("Preamble is always emitted", func(t *testing.T) {
		output := applyFilter(t, dump, nil, []string{"users"})

		assert.Contains(t, output, "SET NAMES utf8mb4;")
	})

	t.Run("Epilogue is always emitted", func(t *testing.T) {
		output := applyFilter(t, dump, nil, []string{"users"})

		assert.Contains(t, output, "-- Dump completed on 2026-01-01")
	})
}

func Test_NewTableFilterReader_IncludeOverridesExclude(t *testing.T) {
	dump := buildDump()

	t.Run("When the same table appears in both include and exclude, include wins", func(t *testing.T) {
		output := applyFilter(t, dump, []string{"users"}, []string{"users"})

		assert.Contains(t, output, "CREATE TABLE `users`")
		assert.Contains(t, output, "INSERT INTO `users` VALUES")
	})

	t.Run("When include is set, exclude is completely ignored", func(t *testing.T) {
		// include=users, exclude=orders — orders should still be dropped (not emitted),
		// because once include is active the exclude list has no effect.
		output := applyFilter(t, dump, []string{"users"}, []string{"orders"})

		assert.Contains(t, output, "CREATE TABLE `users`")
		assert.NotContains(t, output, "CREATE TABLE `orders`")
	})
}

func Test_NewTableFilterReader_WhitespaceInTableNames(t *testing.T) {
	dump := buildDump()

	t.Run("Leading and trailing whitespace around table names in include list is trimmed", func(t *testing.T) {
		output := applyFilter(t, dump, []string{"  users  "}, nil)

		assert.Contains(t, output, "CREATE TABLE `users`")
		assert.NotContains(t, output, "CREATE TABLE `orders`")
	})

	t.Run("Empty strings in include list are silently ignored", func(t *testing.T) {
		output := applyFilter(t, dump, []string{"", "users", ""}, nil)

		assert.Contains(t, output, "CREATE TABLE `users`")
		assert.NotContains(t, output, "CREATE TABLE `orders`")
	})

	t.Run("Whitespace-only entries in exclude list are silently ignored", func(t *testing.T) {
		output := applyFilter(t, dump, nil, []string{"   ", "orders"})

		assert.Contains(t, output, "CREATE TABLE `users`")
		assert.NotContains(t, output, "CREATE TABLE `orders`")
	})
}

func Test_MakeTableSet(t *testing.T) {
	t.Run("Empty slice returns empty map", func(t *testing.T) {
		result := MakeTableSet(nil)

		assert.Empty(t, result)
	})

	t.Run("Single entry is in the map", func(t *testing.T) {
		result := MakeTableSet([]string{"users"})

		assert.True(t, result["users"])
		assert.False(t, result["orders"])
	})

	t.Run("Multiple entries are all in the map", func(t *testing.T) {
		result := MakeTableSet([]string{"users", "orders", "products"})

		assert.True(t, result["users"])
		assert.True(t, result["orders"])
		assert.True(t, result["products"])
		assert.False(t, result["missing"])
	})

	t.Run("Entries with whitespace are trimmed before insertion", func(t *testing.T) {
		result := MakeTableSet([]string{"  users  ", "\torders\t"})

		assert.True(t, result["users"])
		assert.True(t, result["orders"])
		assert.False(t, result["  users  "])
	})

	t.Run("Empty and whitespace-only entries are dropped", func(t *testing.T) {
		result := MakeTableSet([]string{"", "   ", "users"})

		assert.Len(t, result, 1)
		assert.True(t, result["users"])
	})
}
