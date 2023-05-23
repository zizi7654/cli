package itemarchive

import (
	"os"
	"testing"

	"github.com/cli/cli/v2/internal/tableprinter"
	"github.com/cli/cli/v2/pkg/cmd/project/shared/queries"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func TestNewCmdarchiveItem(t *testing.T) {
	tests := []struct {
		name        string
		cli         string
		wants       archiveItemOpts
		wantsErr    bool
		wantsErrMsg string
	}{
		{
			name:        "missing-id",
			cli:         "",
			wantsErr:    true,
			wantsErrMsg: "required flag(s) \"id\" not set",
		},
		{
			name:        "user-and-org",
			cli:         "--user monalisa --org github --id 123",
			wantsErr:    true,
			wantsErrMsg: "only one of `--user` or `--org` may be used",
		},
		{
			name:        "not-a-number",
			cli:         "x --id 123",
			wantsErr:    true,
			wantsErrMsg: "invalid number: x",
		},
		{
			name: "id",
			cli:  "--id 123",
			wants: archiveItemOpts{
				itemID: "123",
			},
		},
		{
			name: "number",
			cli:  "456 --id 123",
			wants: archiveItemOpts{
				number: 456,
				itemID: "123",
			},
		},
		{
			name: "user",
			cli:  "--user monalisa --id 123",
			wants: archiveItemOpts{
				userOwner: "monalisa",
				itemID:    "123",
			},
		},
		{
			name: "org",
			cli:  "--org github  --id 123",
			wants: archiveItemOpts{
				orgOwner: "github",
				itemID:   "123",
			},
		},
		{
			name: "undo",
			cli:  "--undo  --id 123",
			wants: archiveItemOpts{
				undo:   true,
				itemID: "123",
			},
		},
		{
			name: "json",
			cli:  "--format json --id 123",
			wants: archiveItemOpts{
				format: "json",
				itemID: "123",
			},
		},
	}

	os.Setenv("GH_TOKEN", "auth-token")
	defer os.Unsetenv("GH_TOKEN")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			f := &cmdutil.Factory{
				IOStreams: ios,
			}

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts archiveItemOpts
			cmd := NewCmdArchiveItem(f, func(config archiveItemConfig) error {
				gotOpts = config.opts
				return nil
			})

			cmd.SetArgs(argv)
			_, err = cmd.ExecuteC()
			if tt.wantsErr {
				assert.Error(t, err)
				assert.Equal(t, tt.wantsErrMsg, err.Error())
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, tt.wants.number, gotOpts.number)
			assert.Equal(t, tt.wants.userOwner, gotOpts.userOwner)
			assert.Equal(t, tt.wants.orgOwner, gotOpts.orgOwner)
			assert.Equal(t, tt.wants.itemID, gotOpts.itemID)
			assert.Equal(t, tt.wants.undo, gotOpts.undo)
			assert.Equal(t, tt.wants.format, gotOpts.format)
		})
	}
}

func TestRunArchive_User(t *testing.T) {
	defer gock.Off()
	gock.Observe(gock.DumpRequest)

	// get user ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query UserLogin.*",
			"variables": map[string]interface{}{
				"login": "monalisa",
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"user": map[string]interface{}{
					"id": "an ID",
				},
			},
		})

	// get project ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query UserProject.*",
			"variables": map[string]interface{}{
				"login":       "monalisa",
				"number":      1,
				"firstItems":  0,
				"afterItems":  nil,
				"firstFields": 0,
				"afterFields": nil,
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"user": map[string]interface{}{
					"projectV2": map[string]interface{}{
						"id": "an ID",
					},
				},
			},
		})

	// archive item
	gock.New("https://api.github.com").
		Post("/graphql").
		BodyString(`{"query":"mutation ArchiveProjectItem.*","variables":{"input":{"projectId":"an ID","itemId":"item ID"}}}`).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"archiveProjectV2Item": map[string]interface{}{
					"item": map[string]interface{}{
						"id": "item ID",
					},
				},
			},
		})

	client := queries.NewTestClient()

	ios, _, stdout, _ := iostreams.Test()
	config := archiveItemConfig{
		tp: tableprinter.New(ios),
		opts: archiveItemOpts{
			userOwner: "monalisa",
			number:    1,
			itemID:    "item ID",
		},
		client: client,
	}

	err := runArchiveItem(config)
	assert.NoError(t, err)
	assert.Equal(
		t,
		"Archived item\n",
		stdout.String())
}

func TestRunArchive_Org(t *testing.T) {
	defer gock.Off()
	gock.Observe(gock.DumpRequest)
	// get org ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query OrgLogin.*",
			"variables": map[string]interface{}{
				"login": "github",
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"organization": map[string]interface{}{
					"id": "an ID",
				},
			},
		})

	// get project ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query OrgProject.*",
			"variables": map[string]interface{}{
				"login":       "github",
				"number":      1,
				"firstItems":  0,
				"afterItems":  nil,
				"firstFields": 0,
				"afterFields": nil,
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"organization": map[string]interface{}{
					"projectV2": map[string]interface{}{
						"id": "an ID",
					},
				},
			},
		})

	// archive item
	gock.New("https://api.github.com").
		Post("/graphql").
		BodyString(`{"query":"mutation ArchiveProjectItem.*","variables":{"input":{"projectId":"an ID","itemId":"item ID"}}}`).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"archiveProjectV2Item": map[string]interface{}{
					"item": map[string]interface{}{
						"id": "item ID",
					},
				},
			},
		})

	client := queries.NewTestClient()

	ios, _, stdout, _ := iostreams.Test()
	config := archiveItemConfig{
		tp: tableprinter.New(ios),
		opts: archiveItemOpts{
			orgOwner: "github",
			number:   1,
			itemID:   "item ID",
		},
		client: client,
	}

	err := runArchiveItem(config)
	assert.NoError(t, err)
	assert.Equal(
		t,
		"Archived item\n",
		stdout.String())
}

func TestRunArchive_Me(t *testing.T) {
	defer gock.Off()
	gock.Observe(gock.DumpRequest)
	// get viewer ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query ViewerLogin.*",
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"viewer": map[string]interface{}{
					"id": "an ID",
				},
			},
		})

	// get project ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query ViewerProject.*",
			"variables": map[string]interface{}{
				"number":      1,
				"firstItems":  0,
				"afterItems":  nil,
				"firstFields": 0,
				"afterFields": nil,
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"viewer": map[string]interface{}{
					"projectV2": map[string]interface{}{
						"id": "an ID",
					},
				},
			},
		})

	// archive item
	gock.New("https://api.github.com").
		Post("/graphql").
		BodyString(`{"query":"mutation ArchiveProjectItem.*","variables":{"input":{"projectId":"an ID","itemId":"item ID"}}}`).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"archiveProjectV2Item": map[string]interface{}{
					"item": map[string]interface{}{
						"id": "item ID",
					},
				},
			},
		})

	client := queries.NewTestClient()

	ios, _, stdout, _ := iostreams.Test()
	config := archiveItemConfig{
		tp: tableprinter.New(ios),
		opts: archiveItemOpts{
			userOwner: "@me",
			number:    1,
			itemID:    "item ID",
		},
		client: client,
	}

	err := runArchiveItem(config)
	assert.NoError(t, err)
	assert.Equal(
		t,
		"Archived item\n",
		stdout.String())
}

func TestRunArchive_User_Undo(t *testing.T) {
	defer gock.Off()
	gock.Observe(gock.DumpRequest)
	// get user ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query UserLogin.*",
			"variables": map[string]interface{}{
				"login": "monalisa",
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"user": map[string]interface{}{
					"id": "an ID",
				},
			},
		})

	// get project ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query UserProject.*",
			"variables": map[string]interface{}{
				"login":       "monalisa",
				"number":      1,
				"firstItems":  0,
				"afterItems":  nil,
				"firstFields": 0,
				"afterFields": nil,
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"user": map[string]interface{}{
					"projectV2": map[string]interface{}{
						"id": "an ID",
					},
				},
			},
		})

	// archive item
	gock.New("https://api.github.com").
		Post("/graphql").
		BodyString(`{"query":"mutation UnarchiveProjectItem.*","variables":{"input":{"projectId":"an ID","itemId":"item ID"}}}`).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"unarchiveProjectV2Item": map[string]interface{}{
					"item": map[string]interface{}{
						"id": "item ID",
					},
				},
			},
		})

	client := queries.NewTestClient()
	ios, _, stdout, _ := iostreams.Test()
	config := archiveItemConfig{
		tp: tableprinter.New(ios),
		opts: archiveItemOpts{
			userOwner: "monalisa",
			number:    1,
			itemID:    "item ID",
			undo:      true,
		},
		client: client,
	}

	err := runArchiveItem(config)
	assert.NoError(t, err)
	assert.Equal(
		t,
		"Unarchived item\n",
		stdout.String())
}

func TestRunArchive_Org_Undo(t *testing.T) {
	defer gock.Off()
	gock.Observe(gock.DumpRequest)
	// get org ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query OrgLogin.*",
			"variables": map[string]interface{}{
				"login": "github",
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"organization": map[string]interface{}{
					"id": "an ID",
				},
			},
		})

	// get project ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query OrgProject.*",
			"variables": map[string]interface{}{
				"login":       "github",
				"number":      1,
				"firstItems":  0,
				"afterItems":  nil,
				"firstFields": 0,
				"afterFields": nil,
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"organization": map[string]interface{}{
					"projectV2": map[string]interface{}{
						"id": "an ID",
					},
				},
			},
		})

	// archive item
	gock.New("https://api.github.com").
		Post("/graphql").
		BodyString(`{"query":"mutation UnarchiveProjectItem.*","variables":{"input":{"projectId":"an ID","itemId":"item ID"}}}`).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"unarchiveProjectV2Item": map[string]interface{}{
					"item": map[string]interface{}{
						"id": "item ID",
					},
				},
			},
		})

	client := queries.NewTestClient()

	ios, _, stdout, _ := iostreams.Test()
	config := archiveItemConfig{
		tp: tableprinter.New(ios),
		opts: archiveItemOpts{
			orgOwner: "github",
			number:   1,
			itemID:   "item ID",
			undo:     true,
		},
		client: client,
	}

	err := runArchiveItem(config)
	assert.NoError(t, err)
	assert.Equal(
		t,
		"Unarchived item\n",
		stdout.String())
}

func TestRunArchive_Me_Undo(t *testing.T) {
	defer gock.Off()
	gock.Observe(gock.DumpRequest)
	// get viewer ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query ViewerLogin.*",
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"viewer": map[string]interface{}{
					"id": "an ID",
				},
			},
		})

	// get project ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query ViewerProject.*",
			"variables": map[string]interface{}{
				"number":      1,
				"firstItems":  0,
				"afterItems":  nil,
				"firstFields": 0,
				"afterFields": nil,
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"viewer": map[string]interface{}{
					"projectV2": map[string]interface{}{
						"id": "an ID",
					},
				},
			},
		})

	// archive item
	gock.New("https://api.github.com").
		Post("/graphql").
		BodyString(`{"query":"mutation UnarchiveProjectItem.*","variables":{"input":{"projectId":"an ID","itemId":"item ID"}}}`).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"unarchiveProjectV2Item": map[string]interface{}{
					"item": map[string]interface{}{
						"id": "item ID",
					},
				},
			},
		})

	client := queries.NewTestClient()
	ios, _, stdout, _ := iostreams.Test()
	config := archiveItemConfig{
		tp: tableprinter.New(ios),
		opts: archiveItemOpts{
			userOwner: "@me",
			number:    1,
			itemID:    "item ID",
			undo:      true,
		},
		client: client,
	}

	err := runArchiveItem(config)
	assert.NoError(t, err)
	assert.Equal(
		t,
		"Unarchived item\n",
		stdout.String())
}