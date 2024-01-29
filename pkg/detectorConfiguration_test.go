package duck

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGDCConfiguration_MatchingHost(t *testing.T) {
	// Save original hostname
	originalHostname, err := os.Hostname()
	require.NoError(t, err)

	tests := []struct {
		name         string
		gdcs         []GDCConfiguration
		searchName   string
		testHostname string
		wantErr      bool
		wantGDC      *GDCConfiguration
	}{
		{
			name: "single matching GDC",
			gdcs: []GDCConfiguration{
				{
					ID:   1,
					Name: "gdc1",
					Host: originalHostname,
					IP:   "192.168.1.1",
				},
			},
			searchName:   "gdc1",
			testHostname: originalHostname,
			wantErr:      false,
			wantGDC: &GDCConfiguration{
				ID:   1,
				Name: "gdc1",
				Host: originalHostname,
				IP:   "192.168.1.1",
			},
		},
		{
			name: "multiple GDCs, one matches",
			gdcs: []GDCConfiguration{
				{ID: 1, Name: "gdc1", Host: "otherhost", IP: "192.168.1.1"},
				{ID: 2, Name: "gdc2", Host: originalHostname, IP: "192.168.1.2"},
				{ID: 3, Name: "gdc3", Host: "thirdhost", IP: "192.168.1.3"},
			},
			searchName:   "gdc2",
			testHostname: originalHostname,
			wantErr:      false,
			wantGDC: &GDCConfiguration{
				ID:   2,
				Name: "gdc2",
				Host: originalHostname,
				IP:   "192.168.1.2",
			},
		},
		{
			name: "no matching hostname",
			gdcs: []GDCConfiguration{
				{ID: 1, Name: "gdc1", Host: "otherhost", IP: "192.168.1.1"},
			},
			searchName:   "gdc1",
			testHostname: originalHostname,
			wantErr:      true,
			wantGDC:      nil,
		},
		{
			name: "matching hostname but wrong name",
			gdcs: []GDCConfiguration{
				{ID: 1, Name: "gdc1", Host: originalHostname, IP: "192.168.1.1"},
			},
			searchName:   "gdc2",
			testHostname: originalHostname,
			wantErr:      true,
			wantGDC:      nil,
		},
		{
			name:         "empty GDC list",
			gdcs:         []GDCConfiguration{},
			searchName:   "gdc1",
			testHostname: originalHostname,
			wantErr:      true,
			wantGDC:      nil,
		},
		{
			name: "multiple GDCs with same hostname and different names",
			gdcs: []GDCConfiguration{
				{ID: 1, Name: "gdc1", Host: originalHostname, IP: "192.168.1.1"},
				{ID: 2, Name: "gdc2", Host: originalHostname, IP: "192.168.1.2"},
				{ID: 3, Name: "gdc3", Host: originalHostname, IP: "192.168.1.3"},
			},
			searchName:   "gdc2",
			testHostname: originalHostname,
			wantErr:      false,
			wantGDC: &GDCConfiguration{
				ID:   2,
				Name: "gdc2",
				Host: originalHostname,
				IP:   "192.168.1.2",
			},
		},
		{
			name: "empty name string",
			gdcs: []GDCConfiguration{
				{ID: 1, Name: "", Host: originalHostname, IP: "192.168.1.1"},
			},
			searchName:   "",
			testHostname: originalHostname,
			wantErr:      false,
			wantGDC: &GDCConfiguration{
				ID:   1,
				Name: "",
				Host: originalHostname,
				IP:   "192.168.1.1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetGDCConfiguration(tt.gdcs, tt.searchName)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				assert.Contains(t, err.Error(), "GDC not found in configuration")
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantGDC.ID, got.ID)
			assert.Equal(t, tt.wantGDC.Name, got.Name)
			assert.Equal(t, tt.wantGDC.Host, got.Host)
			assert.Equal(t, tt.wantGDC.IP, got.IP)
		})
	}
}

func TestGetLDCConfiguration_MatchingHost(t *testing.T) {
	// Get original hostname
	originalHostname, err := os.Hostname()
	require.NoError(t, err)

	tests := []struct {
		name    string
		ldcs    []LDCConfiguration
		wantErr bool
		wantLDC *LDCConfiguration
	}{
		{
			name: "single matching LDC",
			ldcs: []LDCConfiguration{
				{
					ID:   1,
					Name: "ldc1",
					Host: originalHostname,
					IP:   "192.168.1.1",
				},
			},
			wantErr: false,
			wantLDC: &LDCConfiguration{
				ID:   1,
				Name: "ldc1",
				Host: originalHostname,
				IP:   "192.168.1.1",
			},
		},
		{
			name: "multiple LDCs, one matches",
			ldcs: []LDCConfiguration{
				{ID: 1, Name: "ldc1", Host: "otherhost", IP: "192.168.1.1"},
				{ID: 2, Name: "ldc2", Host: originalHostname, IP: "192.168.1.2"},
			},
			wantErr: false,
			wantLDC: &LDCConfiguration{
				ID:   2,
				Name: "ldc2",
				Host: originalHostname,
				IP:   "192.168.1.2",
			},
		},
		{
			name: "no matching hostname",
			ldcs: []LDCConfiguration{
				{ID: 1, Name: "ldc1", Host: "otherhost", IP: "192.168.1.1"},
			},
			wantErr: true,
			wantLDC: nil,
		},
		{
			name:    "empty LDC list",
			ldcs:    []LDCConfiguration{},
			wantErr: true,
			wantLDC: nil,
		},
		{
			name: "multiple LDCs with same hostname - returns first match",
			ldcs: []LDCConfiguration{
				{ID: 1, Name: "ldc1", Host: originalHostname, IP: "192.168.1.1"},
				{ID: 2, Name: "ldc2", Host: originalHostname, IP: "192.168.1.2"},
				{ID: 3, Name: "ldc3", Host: originalHostname, IP: "192.168.1.3"},
			},
			wantErr: false,
			wantLDC: &LDCConfiguration{
				ID:   1,
				Name: "ldc1",
				Host: originalHostname,
				IP:   "192.168.1.1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetLDCConfiguration(tt.ldcs, "")

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantLDC.ID, got.ID)
			assert.Equal(t, tt.wantLDC.Name, got.Name)
			assert.Equal(t, tt.wantLDC.Host, got.Host)
		})
	}
}

func TestGetEquipmentConfiguration(t *testing.T) {
	eq1 := Equipment{ID: 1, Type: 22, Enabled: true}
	eq2 := Equipment{ID: 2, Type: 22, Enabled: true}
	eq3 := Equipment{ID: 3, Type: 22, Enabled: true}

	ldc1 := LDCConfiguration{
		ID:         1,
		Equipments: []Equipment{eq1, eq2},
	}

	ldc2 := LDCConfiguration{
		ID:         2,
		Equipments: []Equipment{eq3},
	}

	ldcs := []LDCConfiguration{ldc1, ldc2}

	tests := []struct {
		name        string
		ldcs        []LDCConfiguration
		equipmentID int
		wantErr     bool
		wantEqID    int
		wantEnabled bool
	}{
		{
			name:        "equipment in first LDC",
			ldcs:        ldcs,
			equipmentID: 1,
			wantErr:     false,
			wantEqID:    1,
			wantEnabled: true,
		},
		{
			name:        "equipment in second LDC",
			ldcs:        ldcs,
			equipmentID: 3,
			wantErr:     false,
			wantEqID:    3,
			wantEnabled: true,
		},
		{
			name:        "equipment not found",
			ldcs:        ldcs,
			equipmentID: 999,
			wantErr:     true,
		},
		{
			name:        "empty LDC list",
			ldcs:        []LDCConfiguration{},
			equipmentID: 1,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetEquipmentConfiguration(tt.ldcs, tt.equipmentID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				assert.Contains(t, err.Error(), "equipment not found in configuration")
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantEqID, got.ID)
			assert.Equal(t, tt.wantEnabled, got.Enabled)
		})
	}
}

func TestGetEquipmentConfiguration_EnabledField(t *testing.T) {
	tests := []struct {
		name        string
		ldcs        []LDCConfiguration
		equipmentID int
		wantEnabled bool
		wantErr     bool
	}{
		{
			name: "enabled equipment",
			ldcs: []LDCConfiguration{
				{
					ID: 1,
					Equipments: []Equipment{
						{ID: 1, Type: 22, Enabled: true},
					},
				},
			},
			equipmentID: 1,
			wantEnabled: true,
			wantErr:     false,
		},
		{
			name: "disabled equipment",
			ldcs: []LDCConfiguration{
				{
					ID: 1,
					Equipments: []Equipment{
						{ID: 1, Type: 22, Enabled: false},
					},
				},
			},
			equipmentID: 1,
			wantEnabled: false,
			wantErr:     false,
		},
		{
			name: "mixed enabled and disabled equipments",
			ldcs: []LDCConfiguration{
				{
					ID: 1,
					Equipments: []Equipment{
						{ID: 1, Type: 22, Enabled: true},
						{ID: 2, Type: 22, Enabled: false},
						{ID: 3, Type: 22, Enabled: true},
					},
				},
			},
			equipmentID: 2,
			wantEnabled: false,
			wantErr:     false,
		},
		{
			name: "equipment with zero values",
			ldcs: []LDCConfiguration{
				{
					ID: 1,
					Equipments: []Equipment{
						{ID: 1, Type: 0, Enabled: false, DeviceIP: "", HostIP: "", HostPort: 0},
					},
				},
			},
			equipmentID: 1,
			wantEnabled: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetEquipmentConfiguration(tt.ldcs, tt.equipmentID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantEnabled, got.Enabled, "Equipment Enabled field should match expected value")
		})
	}
}

func TestEnabledGDCs(t *testing.T) {
	tests := []struct {
		name      string
		gdcs      []GDCConfiguration
		wantCount int
	}{
		{
			name: "all enabled",
			gdcs: []GDCConfiguration{
				{ID: 1, Enabled: true},
				{ID: 2, Enabled: true},
			},
			wantCount: 2,
		},
		{
			name: "some disabled",
			gdcs: []GDCConfiguration{
				{ID: 1, Enabled: true},
				{ID: 2, Enabled: false},
				{ID: 3, Enabled: true},
			},
			wantCount: 2,
		},
		{
			name: "all disabled",
			gdcs: []GDCConfiguration{
				{ID: 1, Enabled: false},
				{ID: 2, Enabled: false},
			},
			wantCount: 0,
		},
		{
			name:      "empty list",
			gdcs:      []GDCConfiguration{},
			wantCount: 0,
		},
		{
			name: "single enabled",
			gdcs: []GDCConfiguration{
				{ID: 1, Enabled: true},
			},
			wantCount: 1,
		},
		{
			name: "single disabled",
			gdcs: []GDCConfiguration{
				{ID: 1, Enabled: false},
			},
			wantCount: 0,
		},
		{
			name: "preserves original data",
			gdcs: []GDCConfiguration{
				{ID: 1, Name: "gdc1", Host: "host1", IP: "192.168.1.1", Enabled: true},
				{ID: 2, Name: "gdc2", Host: "host2", IP: "192.168.1.2", Enabled: false},
				{ID: 3, Name: "gdc3", Host: "host3", IP: "192.168.1.3", Enabled: true},
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnabledGDCs(tt.gdcs)
			assert.Len(t, result, tt.wantCount)

			// Verify all returned GDCs are enabled
			for _, gdc := range result {
				assert.True(t, gdc.Enabled)
			}

			// Verify original data is preserved for enabled items
			if tt.name == "preserves original data" {
				assert.Equal(t, 1, result[0].ID)
				assert.Equal(t, "gdc1", result[0].Name)
				assert.Equal(t, "host1", result[0].Host)
				assert.Equal(t, "192.168.1.1", result[0].IP)

				assert.Equal(t, 3, result[1].ID)
				assert.Equal(t, "gdc3", result[1].Name)
				assert.Equal(t, "host3", result[1].Host)
				assert.Equal(t, "192.168.1.3", result[1].IP)
			}
		})
	}
}

func TestEnabledLDCs(t *testing.T) {
	tests := []struct {
		name      string
		ldcs      []LDCConfiguration
		wantCount int
	}{
		{
			name: "all enabled",
			ldcs: []LDCConfiguration{
				{ID: 1, Enabled: true},
				{ID: 2, Enabled: true},
			},
			wantCount: 2,
		},
		{
			name: "some disabled",
			ldcs: []LDCConfiguration{
				{ID: 1, Enabled: true},
				{ID: 2, Enabled: false},
				{ID: 3, Enabled: true},
			},
			wantCount: 2,
		},
		{
			name: "all disabled",
			ldcs: []LDCConfiguration{
				{ID: 1, Enabled: false},
			},
			wantCount: 0,
		},
		{
			name:      "empty list",
			ldcs:      []LDCConfiguration{},
			wantCount: 0,
		},
		{
			name: "single enabled",
			ldcs: []LDCConfiguration{
				{ID: 1, Enabled: true},
			},
			wantCount: 1,
		},
		{
			name: "single disabled",
			ldcs: []LDCConfiguration{
				{ID: 1, Enabled: false},
			},
			wantCount: 0,
		},
		{
			name: "preserves original data with equipments",
			ldcs: []LDCConfiguration{
				{
					ID:      1,
					Name:    "ldc1",
					Host:    "host1",
					IP:      "192.168.1.1",
					Enabled: true,
					Equipments: []Equipment{
						{ID: 1, Type: 22, Enabled: true},
					},
				},
				{
					ID:      2,
					Name:    "ldc2",
					Host:    "host2",
					IP:      "192.168.1.2",
					Enabled: false,
				},
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnabledLDCs(tt.ldcs)
			assert.Len(t, result, tt.wantCount)

			// Verify all returned LDCs are enabled
			for _, ldc := range result {
				assert.True(t, ldc.Enabled)
			}

			// Verify original data is preserved for enabled items
			if tt.name == "preserves original data with equipments" {
				assert.Equal(t, 1, result[0].ID)
				assert.Equal(t, "ldc1", result[0].Name)
				assert.Equal(t, "host1", result[0].Host)
				assert.Equal(t, "192.168.1.1", result[0].IP)
				assert.Len(t, result[0].Equipments, 1)
				assert.Equal(t, 1, result[0].Equipments[0].ID)
			}
		})
	}
}

func TestAssignEquipmentToLDC(t *testing.T) {
	eq1 := Equipment{ID: 1, LDC_ID: 1}
	eq2 := Equipment{ID: 2, LDC_ID: 1}
	eq3 := Equipment{ID: 3, LDC_ID: 2}

	ldc1 := LDCConfiguration{ID: 1}
	ldc2 := LDCConfiguration{ID: 2}

	ldcs := []LDCConfiguration{ldc1, ldc2}
	equipments := []Equipment{eq1, eq2, eq3}

	// Call the function
	assignEquipmentToLDC(ldcs, equipments)

	// Verify assignments
	assert.Len(t, ldcs[0].Equipments, 2, "LDC 1 should have 2 equipments")
	assert.Len(t, ldcs[1].Equipments, 1, "LDC 2 should have 1 equipment")

	// Verify correct equipment assignments
	assert.Equal(t, 1, ldcs[0].Equipments[0].ID)
	assert.Equal(t, 2, ldcs[0].Equipments[1].ID)
	assert.Equal(t, 3, ldcs[1].Equipments[0].ID)
}

func TestAssignEquipmentToLDC_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		ldcs       []LDCConfiguration
		equipments []Equipment
		wantCounts []int // Expected equipment count for each LDC
	}{
		{
			name: "equipment with non-matching LDC_ID",
			ldcs: []LDCConfiguration{
				{ID: 1},
				{ID: 2},
			},
			equipments: []Equipment{
				{ID: 1, LDC_ID: 1},
				{ID: 2, LDC_ID: 999}, // Non-existent LDC
				{ID: 3, LDC_ID: 2},
			},
			wantCounts: []int{1, 1}, // Equipment with LDC_ID 999 is silently ignored
		},
		{
			name: "empty equipment list",
			ldcs: []LDCConfiguration{
				{ID: 1},
				{ID: 2},
			},
			equipments: []Equipment{},
			wantCounts: []int{0, 0},
		},
		{
			name: "all equipments with non-matching LDC_IDs",
			ldcs: []LDCConfiguration{
				{ID: 1},
			},
			equipments: []Equipment{
				{ID: 1, LDC_ID: 999},
				{ID: 2, LDC_ID: 888},
			},
			wantCounts: []int{0},
		},
		{
			name: "multiple equipments to same LDC",
			ldcs: []LDCConfiguration{
				{ID: 1},
			},
			equipments: []Equipment{
				{ID: 1, LDC_ID: 1},
				{ID: 2, LDC_ID: 1},
				{ID: 3, LDC_ID: 1},
				{ID: 4, LDC_ID: 1},
			},
			wantCounts: []int{4},
		},
		{
			name: "empty LDC list",
			ldcs: []LDCConfiguration{},
			equipments: []Equipment{
				{ID: 1, LDC_ID: 1},
			},
			wantCounts: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function
			assignEquipmentToLDC(tt.ldcs, tt.equipments)

			// Verify equipment counts for each LDC
			for i, ldc := range tt.ldcs {
				assert.Len(t, ldc.Equipments, tt.wantCounts[i], "LDC %d should have %d equipments", i, tt.wantCounts[i])
			}
		})
	}
}
