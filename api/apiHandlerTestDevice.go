package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"connectrpc.com/connect"
	duck "github.com/jmbenlloch/next_duck/pkg"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
	devicepb "github.com/jmbenlloch/next_duck/rpc/control"
	devicepbconnect "github.com/jmbenlloch/next_duck/rpc/control/controlconnect"
)

var deviceControlMutex sync.Mutex
var rpcTimeout = 10 * time.Second

func connectToDeviceRPCServer(ip string, port int) devicepbconnect.RunControlClient {
	baseURL := fmt.Sprintf("http://%s:%d", ip, port)
	return devicepbconnect.NewRunControlClient(
		http.DefaultClient,
		baseURL,
	)
}

// ===== Test Device Management =====

func (s *DuckAPIServer) StartTestDevices(ctx context.Context, req *connect.Request[pb.StartTestDevicesRequest]) (*connect.Response[pb.StartTestDevicesResponse], error) {
	deviceControlMutex.Lock()
	defer deviceControlMutex.Unlock()

	// Fetch LDCs and Equipments using sqlc querier
	ldcsList, err := s.queries.ListLDCs(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	equipmentsList, err := s.queries.ListEquipments(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	var errorMessages []string

	// Start all enabled devices
	for _, ldc := range ldcsList {
		if !ldc.Enabled.Bool {
			continue
		}
		// Use LDC.ID directly as docker hostname uses 1-indexed naming (sim-1-1, sim-1-2, etc.)
		ldcNumber := int(ldc.ID)

		// Get equipments for this LDC and track their index (1-indexed to match docker)
		eqIdx := 1
		for _, equipment := range equipmentsList {
			if equipment.Ldcid.Int32 != ldc.ID {
				continue
			}
			if !equipment.Enabled.Bool {
				eqIdx++
				continue
			}

			// Calculate device ConnectRPC port
			deviceGRPCPort := 50070 + int(equipment.ID) - 1
			deviceName := fmt.Sprintf("device-%d", equipment.ID)
			// Use docker hostname for ConnectRPC connection (sim-{ldcNumber}-{equipmentIndex})
			deviceHostname := fmt.Sprintf("sim-%d-%d", ldcNumber, eqIdx)

			client := connectToDeviceRPCServer(deviceHostname, deviceGRPCPort)
			deviceCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
			defer cancel()

			_, err := client.StartRun(deviceCtx, connect.NewRequest(&devicepb.DuckRequest{}))
			if err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("Error starting %s: %v", deviceName, err))
			}
			eqIdx++
		}
	}

	if len(errorMessages) > 0 {
		errMsg := "Some devices failed to start: " + fmt.Sprint(errorMessages)
		return connect.NewResponse(&pb.StartTestDevicesResponse{
			Success: false,
			Message: errMsg,
		}), nil
	}

	return connect.NewResponse(&pb.StartTestDevicesResponse{
		Success: true,
		Message: "All test devices started successfully",
	}), nil
}

func (s *DuckAPIServer) StopTestDevices(ctx context.Context, req *connect.Request[pb.StopTestDevicesRequest]) (*connect.Response[pb.StopTestDevicesResponse], error) {
	deviceControlMutex.Lock()
	defer deviceControlMutex.Unlock()

	// Fetch LDCs and Equipments using sqlc querier
	ldcsList, err := s.queries.ListLDCs(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	equipmentsList, err := s.queries.ListEquipments(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	var errorMessages []string

	// Stop all enabled devices
	for _, ldc := range ldcsList {
		if !ldc.Enabled.Bool {
			continue
		}
		// Use LDC.ID directly as docker hostname uses 1-indexed naming (sim-1-1, sim-1-2, etc.)
		ldcNumber := int(ldc.ID)

		// Get equipments for this LDC and track their index (1-indexed to match docker)
		eqIdx := 1
		for _, equipment := range equipmentsList {
			if equipment.Ldcid.Int32 != ldc.ID {
				continue
			}
			if !equipment.Enabled.Bool {
				eqIdx++
				continue
			}

			deviceGRPCPort := 50070 + int(equipment.ID) - 1
			deviceName := fmt.Sprintf("device-%d", equipment.ID)
			// Use docker hostname for ConnectRPC connection (sim-{ldcNumber}-{equipmentIndex})
			deviceHostname := fmt.Sprintf("sim-%d-%d", ldcNumber, eqIdx)

			client := connectToDeviceRPCServer(deviceHostname, deviceGRPCPort)
			deviceCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
			defer cancel()

			_, err := client.StopRun(deviceCtx, connect.NewRequest(&devicepb.DuckRequest{}))
			if err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("Error stopping %s: %v", deviceName, err))
			}
			eqIdx++
		}
	}

	if len(errorMessages) > 0 {
		errMsg := "Some devices failed to stop: " + fmt.Sprint(errorMessages)
		return connect.NewResponse(&pb.StopTestDevicesResponse{
			Success: false,
			Message: errMsg,
		}), nil
	}

	return connect.NewResponse(&pb.StopTestDevicesResponse{
		Success: true,
		Message: "All test devices stopped successfully",
	}), nil
}

func (s *DuckAPIServer) GetTestDevicesStates(ctx context.Context, req *connect.Request[pb.GetTestDevicesStatesRequest]) (*connect.Response[pb.GetTestDevicesStatesResponse], error) {
	// Fetch LDCs and Equipments using sqlc querier
	ldcsList, err := s.queries.ListLDCs(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	equipmentsList, err := s.queries.ListEquipments(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	deviceStates := make(map[string]string)

	// Get state of all enabled devices
	for _, ldc := range ldcsList {
		if !ldc.Enabled.Bool {
			continue
		}
		// Use LDC.ID directly as docker hostname uses 1-indexed naming (sim-1-1, sim-1-2, etc.)
		ldcNumber := int(ldc.ID)

		// Get equipments for this LDC and track their index (1-indexed to match docker)
		eqIdx := 1
		for _, equipment := range equipmentsList {
			if equipment.Ldcid.Int32 != ldc.ID {
				continue
			}
			if !equipment.Enabled.Bool {
				eqIdx++
				continue
			}

			deviceGRPCPort := 50070 + int(equipment.ID) - 1
			deviceName := fmt.Sprintf("device-%d", equipment.ID)
			// Use docker hostname for ConnectRPC connection (sim-{ldcNumber}-{equipmentIndex})
			deviceHostname := fmt.Sprintf("sim-%d-%d", ldcNumber, eqIdx)

			client := connectToDeviceRPCServer(deviceHostname, deviceGRPCPort)
			deviceCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
			defer cancel()

			response, err := client.GetState(deviceCtx, connect.NewRequest(&devicepb.DuckRequest{}))
			if err != nil {
				deviceStates[deviceName] = "ERROR: " + err.Error()
			} else {
				deviceStates[deviceName] = response.Msg.Message
			}
			eqIdx++
		}
	}

	return connect.NewResponse(&pb.GetTestDevicesStatesResponse{
		States: deviceStates,
	}), nil
}

func (s *DuckAPIServer) ListTestDevices(ctx context.Context, req *connect.Request[pb.ListTestDevicesRequest]) (*connect.Response[pb.ListTestDevicesResponse], error) {
	// Fetch LDCs and Equipments using sqlc querier
	ldcsList, err := s.queries.ListLDCs(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	equipmentsList, err := s.queries.ListEquipments(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	// Build LDC configurations from database results
	ldcs := make([]duck.LDCConfiguration, 0, len(ldcsList))
	for _, ldc := range ldcsList {
		ldcConfig := duck.LDCConfiguration{
			ID:             int(ldc.ID),
			IP:             ldc.Ip.String,
			GRPCPort:       int(ldc.GrpcPort.Int32),
			PrometheusPort: int(ldc.PrometheusPort.Int32),
			Name:           ldc.Name.String,
			Host:           ldc.Hostname.String,
			Enabled:        ldc.Enabled.Bool,
		}
		ldcs = append(ldcs, ldcConfig)
	}

	// Assign equipments to LDCs
	for i := range ldcs {
		for _, eq := range equipmentsList {
			if eq.Ldcid.Int32 == int32(ldcs[i].ID) {
				eqConfig := duck.Equipment{
					ID:       int(eq.ID),
					Type:     int(eq.Type.Int32),
					DeviceIP: eq.DeviceIp.String,
					HostIP:   eq.HostIp.String,
					HostPort: int(eq.HostPort.Int32),
					Enabled:  eq.Enabled.Bool,
				}
				ldcs[i].Equipments = append(ldcs[i].Equipments, eqConfig)
			}
		}
	}

	devices := []*pb.TestDevice{}

	for _, ldc := range ldcs {
		for _, equipment := range ldc.Equipments {
			device := &pb.TestDevice{
				EquipmentId:        int32(equipment.ID),
				DeviceName:         fmt.Sprintf("device-%d", equipment.ID),
				HostIp:             equipment.DeviceIP,
				HostPort:           int32(equipment.HostPort),
				GrpcPort:           int32(50070 + equipment.ID - 1),
				PrometheusPort:     int32(12130 + equipment.ID - 1),
				RateHz:             0.1,
				PacketsPerEvent:    85,
				PacketSize:         500,
				ErrorInjectionRate: 0.0,
				MaxEvents:          0,
				Enabled:            equipment.Enabled,
			}

			// Try to find simulator-specific configuration using sqlc
			simParams, err := s.queries.GetSimulatorParams(ctx, int32(equipment.ID))
			if err == nil {
				device.RateHz = simParams.Replayrate
				device.PacketSize = simParams.Packetsize
				device.MaxEvents = simParams.Maxevents
			}

			devices = append(devices, device)
		}
	}

	return connect.NewResponse(&pb.ListTestDevicesResponse{
		Devices: devices,
	}), nil
}

func (s *DuckAPIServer) GetTestDeviceStatistics(ctx context.Context, req *connect.Request[pb.GetTestDeviceStatisticsRequest]) (*connect.Response[pb.GetTestDeviceStatisticsResponse], error) {
	deviceID := int(req.Msg.DeviceId)

	// Fetch LDCs and Equipments using sqlc querier
	ldcsList, err := s.queries.ListLDCs(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	equipmentsList, err := s.queries.ListEquipments(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	// Find the equipment and derive its hostname
	var deviceHostname string
	found := false

	for _, ldc := range ldcsList {
		ldcNumber := int(ldc.ID) // Use 1-indexed to match docker hostname
		for _, equipment := range equipmentsList {
			if equipment.Ldcid.Int32 == ldc.ID && int(equipment.ID) == deviceID {
				// Find equipment index within this LDC (1-indexed to match docker)
				eqIdx := 1
				for _, eq := range equipmentsList {
					if eq.Ldcid.Int32 == ldc.ID {
						if int(eq.ID) == deviceID {
							break
						}
						eqIdx++
					}
				}
				deviceHostname = fmt.Sprintf("sim-%d-%d", ldcNumber, eqIdx)
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("device not found"))
	}

	deviceGRPCPort := 50070 + deviceID - 1

	client := connectToDeviceRPCServer(deviceHostname, deviceGRPCPort)
	deviceCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	stats, err := client.GetRunStatistics(deviceCtx, connect.NewRequest(&devicepb.DuckRequest{}))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("error getting statistics: %w", err))
	}

	return connect.NewResponse(&pb.GetTestDeviceStatisticsResponse{
		Statistics: &pb.TestDeviceStatistics{
			DeviceId: int32(deviceID),
			Events:   stats.Msg.Events,
			Bytes:    stats.Msg.Bytes,
			Errors:   stats.Msg.Errors,
		},
	}), nil
}
