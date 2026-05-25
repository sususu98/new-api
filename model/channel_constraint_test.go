package model

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	kitdto "github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestFilterCandidateIDs(t *testing.T) {
	alphaSetting := `{"task_plugin_key":"alpha"}`
	betaSetting := `{"task_plugin_key":"beta"}`
	alpha := &Channel{Id: 900001, Type: constant.ChannelTypeTaskPlugin, Status: common.ChannelStatusEnabled, Setting: &alphaSetting}
	beta := &Channel{Id: 900002, Type: constant.ChannelTypeTaskPlugin, Status: common.ChannelStatusEnabled, Setting: &betaSetting}
	ordinary := &Channel{Id: 900003, Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled}
	kling := &Channel{Id: 900004, Type: constant.ChannelTypeKling, Status: common.ChannelStatusEnabled}
	jimeng := &Channel{Id: 900005, Type: constant.ChannelTypeJimeng, Status: common.ChannelStatusEnabled}
	matchingCustom := &Channel{Id: 900010, Type: constant.ChannelTypeAdvancedCustom, Status: common.ChannelStatusEnabled}
	matchingCustom.SetOtherSettings(kitdto.ChannelOtherSettings{
		AdvancedCustom: &kitdto.AdvancedCustomConfig{
			Routes: []kitdto.AdvancedCustomRoute{{
				IncomingPath: "/v1/chat/completions",
				Models:       []string{"gpt-4"},
			}},
		},
	})
	otherCustom := &Channel{Id: 900011, Type: constant.ChannelTypeAdvancedCustom, Status: common.ChannelStatusEnabled}
	otherCustom.SetOtherSettings(kitdto.ChannelOtherSettings{
		AdvancedCustom: &kitdto.AdvancedCustomConfig{
			Routes: []kitdto.AdvancedCustomRoute{{
				IncomingPath: "/v1/responses",
				Models:       []string{"gpt-4"},
			}},
		},
	})

	pathFilter := dto.ChannelFilter{Kind: dto.FilterRequestPath, RequestPath: "/v1/chat/completions"}
	emptyPathFilter := dto.ChannelFilter{Kind: dto.FilterRequestPath, RequestPath: ""}

	tests := []struct {
		name      string
		ids       []int
		modelName string
		filters   []dto.ChannelFilter
		wantKept  []int
		wantEmpty dto.ChannelFilterKind
	}{
		{
			name:      "identity keeps matching type-59 key",
			ids:       []int{900001, 900002},
			modelName: "shared",
			filters:   identityFilters("alpha", nil),
			wantKept:  []int{900001},
		},
		{
			name:      "identity empty key drops all type-59",
			ids:       []int{900001, 900002},
			modelName: "shared",
			filters:   identityFilters("", nil),
			wantKept:  []int{},
			wantEmpty: dto.FilterTaskPluginIdentity,
		},
		{
			name:      "identity empty key keeps ordinary channel",
			ids:       []int{900003},
			modelName: "ordinary",
			filters:   identityFilters("", nil),
			wantKept:  []int{900003},
		},
		{
			name:      "identity keeps matching legacy type",
			ids:       []int{900004, 900005},
			modelName: "legacy",
			filters:   identityFilters("legacy-alpha", []int{constant.ChannelTypeKling}),
			wantKept:  []int{900004},
		},
		{
			name:      "identity keeps all listed legacy types",
			ids:       []int{900004, 900005},
			modelName: "legacy",
			filters:   identityFilters("legacy-alpha", []int{constant.ChannelTypeKling, constant.ChannelTypeJimeng}),
			wantKept:  []int{900004, 900005},
		},
		{
			name:      "identity keyed with no types drops legacy",
			ids:       []int{900004, 900005},
			modelName: "legacy",
			filters:   identityFilters("legacy-alpha", nil),
			wantKept:  []int{},
			wantEmpty: dto.FilterTaskPluginIdentity,
		},
		{
			name:      "identity drops missing cache entry",
			ids:       []int{900004, 999999},
			modelName: "legacy",
			filters:   identityFilters("legacy-alpha", []int{constant.ChannelTypeKling}),
			wantKept:  []int{900004},
		},
		{
			name:      "empty request path is a passthrough including missing ids",
			ids:       []int{900003, 900010, 999999},
			modelName: "gpt-4",
			filters:   []dto.ChannelFilter{emptyPathFilter},
			wantKept:  []int{900003, 900010, 999999},
		},
		{
			name:      "request path keeps missing cache entry for consistency",
			ids:       []int{900003, 999999},
			modelName: "gpt-4",
			filters:   []dto.ChannelFilter{pathFilter},
			wantKept:  []int{900003, 999999},
		},
		{
			name:      "request path keeps matching type-58 and ordinary",
			ids:       []int{900003, 900010, 900011},
			modelName: "gpt-4",
			filters:   []dto.ChannelFilter{pathFilter},
			wantKept:  []int{900003, 900010},
		},
		{
			name:      "request path empties when only unmatched type-58 remains",
			ids:       []int{900011},
			modelName: "gpt-4",
			filters:   []dto.ChannelFilter{pathFilter},
			wantKept:  []int{},
			wantEmpty: dto.FilterRequestPath,
		},
		{
			name:      "intersection attributes empty set to identity after path keeps candidates",
			ids:       []int{900001, 900010},
			modelName: "gpt-4",
			filters:   []dto.ChannelFilter{pathFilter, identityFilters("missing", nil)[0]},
			wantKept:  []int{},
			wantEmpty: dto.FilterTaskPluginIdentity,
		},
		{
			name:      "intersection attributes empty set to path when path runs first",
			ids:       []int{900011},
			modelName: "gpt-4",
			filters:   []dto.ChannelFilter{identityFilters("", nil)[0], pathFilter},
			wantKept:  []int{},
			wantEmpty: dto.FilterRequestPath,
		},
	}

	channelSyncLock.Lock()
	previous := channelsIDM
	channelsIDM = map[int]*Channel{
		900001: alpha,
		900002: beta,
		900003: ordinary,
		900004: kling,
		900005: jimeng,
		900010: matchingCustom,
		900011: otherCustom,
	}
	t.Cleanup(func() {
		channelsIDM = previous
		channelSyncLock.Unlock()
	})

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			kept, emptiedBy := filterCandidateIDs(testCase.ids, testCase.modelName, testCase.filters)
			if testCase.wantKept == nil {
				assert.Nil(t, kept)
			} else {
				assert.Equal(t, testCase.wantKept, kept)
			}
			assert.Equal(t, testCase.wantEmpty, emptiedBy)
		})
	}
}

func TestChannelSatisfiesFilters(t *testing.T) {
	alphaSetting := `{"task_plugin_key":"alpha"}`
	alpha := &Channel{Id: 1, Type: constant.ChannelTypeTaskPlugin, Setting: &alphaSetting}
	ordinary := &Channel{Id: 2, Type: constant.ChannelTypeOpenAI}
	custom := &Channel{Id: 3, Type: constant.ChannelTypeAdvancedCustom}
	custom.SetOtherSettings(kitdto.ChannelOtherSettings{
		AdvancedCustom: &kitdto.AdvancedCustomConfig{
			Routes: []kitdto.AdvancedCustomRoute{{
				IncomingPath: "/v1/chat/completions",
				Models:       []string{"gpt-4"},
			}},
		},
	})

	ok, kind := ChannelSatisfiesFilters(nil, "gpt-4", nil)
	assert.False(t, ok)
	assert.Equal(t, dto.ChannelFilterKind(""), kind)

	ok, kind = ChannelSatisfiesFilters(alpha, "shared", identityFilters("alpha", nil))
	require.True(t, ok)
	assert.Equal(t, dto.ChannelFilterKind(""), kind)

	ok, kind = ChannelSatisfiesFilters(alpha, "shared", identityFilters("beta", nil))
	assert.False(t, ok)
	assert.Equal(t, dto.FilterTaskPluginIdentity, kind)

	ok, kind = ChannelSatisfiesFilters(ordinary, "gpt-4", []dto.ChannelFilter{{
		Kind:        dto.FilterRequestPath,
		RequestPath: "/v1/chat/completions",
	}})
	require.True(t, ok)
	assert.Equal(t, dto.ChannelFilterKind(""), kind)

	ok, kind = ChannelSatisfiesFilters(custom, "gpt-4", []dto.ChannelFilter{{
		Kind:        dto.FilterRequestPath,
		RequestPath: "/v1/responses",
	}})
	assert.False(t, ok)
	assert.Equal(t, dto.FilterRequestPath, kind)
}

// All selectors, including pinned and affinity candidates, share this predicate.
func TestResponsesRequestPathConstraints(t *testing.T) {
	custom := &Channel{Type: constant.ChannelTypeAdvancedCustom}
	custom.SetOtherSettings(kitdto.ChannelOtherSettings{
		AdvancedCustom: &kitdto.AdvancedCustomConfig{Routes: []kitdto.AdvancedCustomRoute{
			{IncomingPath: "/v1/responses", Models: []string{"allowed"}},
			{IncomingPath: "/v1/responses/compact", Models: []string{"compact-allowed"}},
		}},
	})
	for _, tc := range []struct {
		name        string
		channel     *Channel
		path, model string
		allowed     bool
	}{
		{"custom responses route", custom, "/v1/responses", "allowed", true},
		{"custom responses wrong model", custom, "/v1/responses", "other", false},
		{"custom compact route", custom, "/v1/responses/compact", "compact-allowed", true},
		{"responses route does not grant compact", custom, "/v1/responses/compact", "allowed", false},
		{"custom missing config", &Channel{Type: constant.ChannelTypeAdvancedCustom}, "/v1/responses", "allowed", false},
		{"vLLM responses", &Channel{Type: constant.ChannelTypeVLLM}, "/v1/responses", "allowed", true},
		{"vLLM no compact route", &Channel{Type: constant.ChannelTypeVLLM}, "/v1/responses/compact", "allowed", false},
		{"SGLang responses", &Channel{Type: constant.ChannelTypeSGLang}, "/v1/responses", "allowed", true},
		{"SGLang no compact route", &Channel{Type: constant.ChannelTypeSGLang}, "/v1/responses/compact", "allowed", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ok, kind := ChannelSatisfiesFilters(tc.channel, tc.model, []dto.ChannelFilter{{Kind: dto.FilterRequestPath, RequestPath: tc.path}})
			assert.Equal(t, tc.allowed, ok)
			if tc.allowed {
				assert.Empty(t, kind)
			} else {
				assert.Equal(t, dto.FilterRequestPath, kind)
			}
		})
	}
}

func TestResponsesChannelSelection(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver gorm.Dialector
			switch dialect {
			case "sqlite":
				driver = sqlite.Open(filepath.Join(t.TempDir(), "endpoints.db"))
			case "mysql":
				dsn := os.Getenv("TEST_MYSQL_DSN")
				if dsn == "" {
					t.Skip("TEST_MYSQL_DSN is not configured")
				}
				driver = mysql.Open(dsn)
			case "postgres":
				dsn := os.Getenv("TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("TEST_POSTGRES_DSN is not configured")
				}
				driver = postgres.Open(dsn)
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
			require.NoError(t, db.AutoMigrate(&Channel{}, &Ability{}))
			var version string
			query := "SELECT version()"
			if dialect == "sqlite" {
				query = "SELECT sqlite_version()"
			}
			require.NoError(t, db.Raw(query).Scan(&version).Error)
			t.Logf("%s version: %s", dialect, version)

			oldDB, oldCache := DB, common.MemoryCacheEnabled
			oldGroupCol := commonGroupCol
			channelSyncLock.Lock()
			oldGroups, oldChannels := group2model2channels, channelsIDM
			group2model2channels = map[string]map[string][]int{}
			channelsIDM = map[int]*Channel{}
			channelSyncLock.Unlock()
			DB = db
			commonGroupCol = "`group`"
			if dialect == "postgres" {
				commonGroupCol = `"group"`
			}
			t.Cleanup(func() {
				DB, common.MemoryCacheEnabled, commonGroupCol = oldDB, oldCache, oldGroupCol
				channelSyncLock.Lock()
				group2model2channels, channelsIDM = oldGroups, oldChannels
				channelSyncLock.Unlock()
			})
			group := "endpoint-selection-test"
			var ids []int
			t.Cleanup(func() {
				require.NoError(t, db.Where("channel_id IN ?", ids).Delete(&Ability{}).Error)
				require.NoError(t, db.Where("id IN ?", ids).Delete(&Channel{}).Error)
			})
			for i, channelType := range []int{constant.ChannelTypeMoonshot, constant.ChannelTypeAnthropic, constant.ChannelTypeOpenAI} {
				priority, weight := int64(30-i*10), uint(100)
				ch := &Channel{Type: channelType, Name: "endpoint-test", Key: "test-key", Status: common.ChannelStatusEnabled, Group: group, Models: "shared", Priority: &priority, Weight: &weight}
				require.NoError(t, db.Create(ch).Error)
				ids = append(ids, ch.Id)
				require.NoError(t, db.Create(&Ability{Group: group, Model: "shared", ChannelId: ch.Id, Enabled: true, Priority: &priority, Weight: weight}).Error)
				if i == 0 {
					require.NoError(t, db.Create(&Ability{Group: group, Model: "unsupported", ChannelId: ch.Id, Enabled: true, Priority: &priority, Weight: weight}).Error)
				}
				channelSyncLock.Lock()
				channelsIDM[ch.Id] = ch
				channelSyncLock.Unlock()
			}
			channelSyncLock.Lock()
			group2model2channels[group] = map[string][]int{"shared": ids, "unsupported": {ids[0]}, "gpt-4-gizmo-*": ids}
			channelSyncLock.Unlock()
			for _, cache := range []bool{false, true} {
				name := "database"
				if cache {
					name = "cache"
				}
				t.Run(name, func(t *testing.T) {
					common.MemoryCacheEnabled = cache
					for _, tc := range []struct {
						name, model, path string
						retry, wantID     int
					}{
						{"responses filters before priority", "shared", "/v1/responses", 0, ids[1]},
						{"responses retry priority", "shared", "/v1/responses", 1, ids[2]},
						{"responses retry clamps", "shared", "/v1/responses", 99, ids[2]},
						{"compact rejects conversion-only", "shared", "/v1/responses/compact", 0, ids[2]},
						{"no compatible channel", "unsupported", "/v1/responses", 0, 0},
						{"chat is unchanged", "shared", "/v1/chat/completions", 0, ids[0]},
						{"path boundary", "shared", "/v1/responsesXYZ", 0, ids[0]},
					} {
						t.Run(tc.name, func(t *testing.T) {
							ch, err := GetRandomSatisfiedChannel(group, tc.model, tc.retry, []dto.ChannelFilter{{Kind: dto.FilterRequestPath, RequestPath: tc.path}})
							require.NoError(t, err)
							if tc.wantID == 0 {
								assert.Nil(t, ch)
								return
							}
							require.NotNil(t, ch)
							assert.Equal(t, tc.wantID, ch.Id)
						})
					}
					if cache {
						ch, err := GetRandomSatisfiedChannel(group, "gpt-4-gizmo-example", 0, []dto.ChannelFilter{{Kind: dto.FilterRequestPath, RequestPath: "/v1/responses"}})
						require.NoError(t, err)
						require.NotNil(t, ch)
						assert.Equal(t, ids[1], ch.Id, "normalized fallback must still apply endpoint filtering")
					}
				})
			}
		})
	}
}
