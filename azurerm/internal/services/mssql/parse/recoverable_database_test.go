package parse

import (
	"testing"
)

func TestMssqlRecoverableDBID(t *testing.T) {
	testData := []struct {
		Name     string
		Input    string
		Expected *MsSqlRecoverableDBId
	}{
		{
			Name:     "Empty",
			Input:    "",
			Expected: nil,
		},
		{
			Name:     "No Resource Groups Segment",
			Input:    "/subscriptions/00000000-0000-0000-0000-000000000000",
			Expected: nil,
		},
		{
			Name:     "No Resource Groups Value",
			Input:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/",
			Expected: nil,
		},
		{
			Name:     "Resource Group ID",
			Input:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/foo/",
			Expected: nil,
		},
		{
			Name:     "Missing Sql Server Value",
			Input:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resGroup1/providers/Microsoft.Sql/servers/",
			Expected: nil,
		},
		{
			Name:     "Missing Sql Recoverable Database",
			Input:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resGroup1/providers/Microsoft.Sql/servers/sqlServer1",
			Expected: nil,
		},
		{
			Name:     "Missing Sql Recoverable Database Value",
			Input:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resGroup1/providers/Microsoft.Sql/servers/sqlServer1/recoverabledatabases",
			Expected: nil,
		},
		{
			Name:  "Sql Database ID",
			Input: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resGroup1/providers/Microsoft.Sql/servers/sqlServer1/recoverabledatabases/sqlDB1",
			Expected: &MsSqlRecoverableDBId{
				Name:          "sqlDB1",
				MsSqlServer:   "sqlServer1",
				ResourceGroup: "resGroup1",
			},
		},
		{
			Name:     "Wrong Casing",
			Input:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resGroup1/providers/Microsoft.Sql/servers/sqlServer1/Recoverabledatabases/sqlDB1",
			Expected: nil,
		},
	}

	for _, v := range testData {
		t.Logf("[DEBUG] Testing %q", v.Name)

		actual, err := MssqlRecoverableDBID(v.Input)
		if err != nil {
			if v.Expected == nil {
				continue
			}

			t.Fatalf("Expected a value but got an error: %s", err)
		}

		if actual.Name != v.Expected.Name {
			t.Fatalf("Expected %q but got %q for Name", v.Expected.Name, actual.Name)
		}

		if actual.MsSqlServer != v.Expected.MsSqlServer {
			t.Fatalf("Expected %q but got %q for Sql Server", v.Expected.Name, actual.Name)
		}

		if actual.ResourceGroup != v.Expected.ResourceGroup {
			t.Fatalf("Expected %q but got %q for Resource Group", v.Expected.ResourceGroup, actual.ResourceGroup)
		}
	}
}
