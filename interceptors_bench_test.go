package apideprecation

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pb "github.com/belo4ya/grpc-api-deprecation/internal/testdata/proto/proto"
)

func BenchmarkUnaryServerInterceptor(b *testing.B) {
	benchmarks := []struct {
		name string
		req  proto.Message
	}{
		{name: "XS", req: requestExtraSmall()},
		{name: "S", req: requestSmall()},
		{name: "M", req: requestMedium()},
		{name: "L", req: requestLarge()},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()
			interceptor := makeInterceptor()
			b.ResetTimer()
			for range b.N {
				if err := interceptor(bb.req); err != nil {
					b.Fatalf("interceptor: %v", err)
				}
			}
		})
	}

	b.Run("cold start", func(b *testing.B) {
		for _, bb := range benchmarks {
			b.Run(bb.name, func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					interceptor := makeInterceptor()
					if err := interceptor(bb.req); err != nil {
						b.Fatalf("interceptor: %v", err)
					}
				}
			})
		}
	})
}

func makeInterceptor() func(req any) error {
	ctx := context.Background()
	interceptor := NewMetrics().UnaryServerInterceptor()
	return func(req any) error {
		_, err := interceptor(
			ctx, req,
			&grpc.UnaryServerInfo{FullMethod: "/benchmark.ClusterService/Create"},
			func(ctx context.Context, req any) (any, error) { return nil, nil },
		)
		return err
	}
}

func requestExtraSmall() proto.Message {
	return &pb.ListUsableSubnetworksRequest{
		Parent:    "test",
		Filter:    "test",
		PageSize:  100,
		PageToken: "test",
	}
}

func requestSmall() proto.Message {
	return &pb.SetMaintenancePolicyRequest{
		ProjectId: "test",
		Zone:      "test",
		ClusterId: "test",
		MaintenancePolicy: &pb.MaintenancePolicy{
			Window: &pb.MaintenanceWindow{
				Policy: &pb.MaintenanceWindow_DailyMaintenanceWindow{
					DailyMaintenanceWindow: &pb.DailyMaintenanceWindow{
						StartTime: "test",
						Duration:  "test",
					},
				},
				MaintenanceExclusions: map[string]*pb.TimeWindow{
					"a": {
						StartTime: timestamppb.Now(),
						EndTime:   timestamppb.Now(),
					},
					"b": {
						StartTime: timestamppb.Now(),
						EndTime:   timestamppb.Now(),
					},
					"c": {
						StartTime: timestamppb.Now(),
						EndTime:   timestamppb.Now(),
					},
				},
			},
			ResourceVersion: "test",
		},
		Name: "test",
	}
}

func requestMedium() *pb.NodeConfig {
	return &pb.NodeConfig{
		MachineType:    "test",
		DiskSizeGb:     100,
		OauthScopes:    sliceStr,
		ServiceAccount: "test",
		Metadata:       mapStr,
		ImageType:      "test",
		Labels:         mapStr,
		LocalSsdCount:  100,
		Tags:           sliceStr,
		Preemptible:    true,
		Accelerators: []*pb.AcceleratorConfig{
			{
				AcceleratorCount: 100,
				AcceleratorType:  "test",
				GpuPartitionSize: "test",
				GpuSharingConfig: &pb.GPUSharingConfig{
					MaxSharedClientsPerGpu: 100,
					GpuSharingStrategy:     lo.ToPtr(pb.GPUSharingConfig_MPS),
				},
				GpuDriverInstallationConfig: &pb.GPUDriverInstallationConfig{
					GpuDriverVersion: lo.ToPtr(pb.GPUDriverInstallationConfig_DEFAULT),
				},
			},
			{
				AcceleratorCount: 100,
				AcceleratorType:  "test",
				GpuPartitionSize: "test",
				GpuSharingConfig: &pb.GPUSharingConfig{
					MaxSharedClientsPerGpu: 100,
					GpuSharingStrategy:     lo.ToPtr(pb.GPUSharingConfig_MPS),
				},
				GpuDriverInstallationConfig: &pb.GPUDriverInstallationConfig{
					GpuDriverVersion: lo.ToPtr(pb.GPUDriverInstallationConfig_DEFAULT),
				},
			},
		},
		DiskType:       "test",
		MinCpuPlatform: "test",
		WorkloadMetadataConfig: &pb.WorkloadMetadataConfig{
			Mode: pb.WorkloadMetadataConfig_GKE_METADATA,
		},
		Taints: []*pb.NodeTaint{{Key: "test", Value: "test", Effect: 100}},
		SandboxConfig: &pb.SandboxConfig{
			Type: pb.SandboxConfig_GVISOR,
		},
		NodeGroup: "test",
		ReservationAffinity: &pb.ReservationAffinity{
			ConsumeReservationType: 100,
			Key:                    "test",
			Values:                 sliceStr,
		},
		ShieldedInstanceConfig: &pb.ShieldedInstanceConfig{
			EnableSecureBoot:          true,
			EnableIntegrityMonitoring: true,
		},
		LinuxNodeConfig: &pb.LinuxNodeConfig{
			Sysctls:    mapStr,
			CgroupMode: 100,
			Hugepages: &pb.LinuxNodeConfig_HugepagesConfig{
				HugepageSize2M: lo.ToPtr[int32](100),
				HugepageSize1G: lo.ToPtr[int32](100),
			},
			TransparentHugepageEnabled: 100,
			TransparentHugepageDefrag:  100,
		},
		KubeletConfig: &pb.NodeKubeletConfig{
			CpuManagerPolicy: "test",
			TopologyManager: &pb.TopologyManager{
				Policy: "test",
				Scope:  "test",
			},
			MemoryManager: &pb.MemoryManager{
				Policy: "test",
			},
			CpuCfsQuota:                        wrapperspb.Bool(true),
			CpuCfsQuotaPeriod:                  "test",
			PodPidsLimit:                       100,
			InsecureKubeletReadonlyPortEnabled: lo.ToPtr(true),
			ImageGcLowThresholdPercent:         100,
			ImageGcHighThresholdPercent:        100,
			ImageMinimumGcAge:                  "test",
			ImageMaximumGcAge:                  "test",
			ContainerLogMaxSize:                "test",
			ContainerLogMaxFiles:               100,
			AllowedUnsafeSysctls:               sliceStr,
			EvictionSoft: &pb.EvictionSignals{
				MemoryAvailable:   "test",
				NodefsAvailable:   "test",
				NodefsInodesFree:  "test",
				ImagefsAvailable:  "test",
				ImagefsInodesFree: "test",
				PidAvailable:      "test",
			},
			EvictionSoftGracePeriod: &pb.EvictionGracePeriod{
				MemoryAvailable:   "test",
				NodefsAvailable:   "test",
				NodefsInodesFree:  "test",
				ImagefsAvailable:  "test",
				ImagefsInodesFree: "test",
				PidAvailable:      "test",
			},
			EvictionMinimumReclaim: &pb.EvictionMinimumReclaim{
				MemoryAvailable:   "test",
				NodefsAvailable:   "test",
				NodefsInodesFree:  "test",
				ImagefsAvailable:  "test",
				ImagefsInodesFree: "test",
				PidAvailable:      "test",
			},
			EvictionMaxPodGracePeriodSeconds: 100,
			MaxParallelImagePulls:            100,
			SingleProcessOomKill:             lo.ToPtr(true),
		},
		BootDiskKmsKey: "test",
		GcfsConfig:     &pb.GcfsConfig{Enabled: true},
		AdvancedMachineFeatures: &pb.AdvancedMachineFeatures{
			ThreadsPerCore:             lo.ToPtr[int64](100),
			EnableNestedVirtualization: lo.ToPtr(true),
			PerformanceMonitoringUnit:  lo.ToPtr(pb.AdvancedMachineFeatures_ENHANCED),
		},
		Gvnic:                          &pb.VirtualNIC{Enabled: true},
		Spot:                           true,
		ConfidentialNodes:              nil,
		FastSocket:                     nil,
		ResourceLabels:                 nil,
		LoggingConfig:                  nil,
		WindowsNodeConfig:              nil,
		LocalNvmeSsdBlockConfig:        nil,
		EphemeralStorageLocalSsdConfig: nil,
	}
}

func requestLarge() proto.Message {
	return &pb.CreateClusterRequest{
		ProjectId: "test",
		Zone:      "test",
		Cluster: &pb.Cluster{
			Name:             "test",
			Description:      "test",
			InitialNodeCount: 100,
			NodeConfig:       requestMedium(),
			MasterAuth: &pb.MasterAuth{
				Username:                "test",
				Password:                "test",
				ClientCertificateConfig: &pb.ClientCertificateConfig{IssueClientCertificate: true},
				ClusterCaCertificate:    "test",
				ClientCertificate:       "test",
				ClientKey:               "test",
			},
			LoggingService:    "test",
			MonitoringService: "test",
			Network:           "test",
			ClusterIpv4Cidr:   "test",
			AddonsConfig: &pb.AddonsConfig{
				HttpLoadBalancing:        &pb.HttpLoadBalancing{Disabled: true},
				HorizontalPodAutoscaling: &pb.HorizontalPodAutoscaling{Disabled: true},
				KubernetesDashboard:      &pb.KubernetesDashboard{Disabled: true},
				NetworkPolicyConfig:      &pb.NetworkPolicyConfig{Disabled: true},
				CloudRunConfig: &pb.CloudRunConfig{
					Disabled:         true,
					LoadBalancerType: pb.CloudRunConfig_LOAD_BALANCER_TYPE_EXTERNAL,
				},
				DnsCacheConfig:                   &pb.DnsCacheConfig{Enabled: true},
				ConfigConnectorConfig:            &pb.ConfigConnectorConfig{Enabled: true},
				GcePersistentDiskCsiDriverConfig: &pb.GcePersistentDiskCsiDriverConfig{Enabled: true},
				GcpFilestoreCsiDriverConfig:      &pb.GcpFilestoreCsiDriverConfig{Enabled: true},
				GkeBackupAgentConfig:             &pb.GkeBackupAgentConfig{Enabled: true},
				GcsFuseCsiDriverConfig:           &pb.GcsFuseCsiDriverConfig{Enabled: true},
				StatefulHaConfig:                 &pb.StatefulHAConfig{Enabled: true},
				ParallelstoreCsiDriverConfig:     &pb.ParallelstoreCsiDriverConfig{Enabled: true},
				RayOperatorConfig: &pb.RayOperatorConfig{
					Enabled:                    true,
					RayClusterLoggingConfig:    &pb.RayClusterLoggingConfig{Enabled: true},
					RayClusterMonitoringConfig: &pb.RayClusterMonitoringConfig{Enabled: true},
				},
				HighScaleCheckpointingConfig: &pb.HighScaleCheckpointingConfig{Enabled: true},
				LustreCsiDriverConfig: &pb.LustreCsiDriverConfig{
					Enabled:                true,
					EnableLegacyLustrePort: true,
				},
			},
			Subnetwork: "test",
			NodePools: []*pb.NodePool{
				{
					Name:             "test",
					Config:           requestMedium(),
					InitialNodeCount: 100,
					Locations:        sliceStr,
					NetworkConfig: &pb.NodeNetworkConfig{
						CreatePodRange:          true,
						PodRange:                "test",
						PodIpv4CidrBlock:        "test",
						PodIpv4RangeUtilization: 100,
						Subnetwork:              "test",
					},
					SelfLink:          "test",
					Version:           "test",
					InstanceGroupUrls: sliceStr,
					Status:            100,
					StatusMessage:     "test",
					Autoscaling: &pb.NodePoolAutoscaling{
						Enabled:           true,
						MinNodeCount:      100,
						MaxNodeCount:      100,
						Autoprovisioned:   true,
						LocationPolicy:    100,
						TotalMinNodeCount: 100,
						TotalMaxNodeCount: 100,
					},
					Management: &pb.NodeManagement{
						AutoUpgrade: true,
						AutoRepair:  true,
						UpgradeOptions: &pb.AutoUpgradeOptions{
							AutoUpgradeStartTime: "test",
							Description:          "test",
						},
					},
					MaxPodsConstraint: &pb.MaxPodsConstraint{MaxPodsPerNode: 100},
					Conditions: []*pb.StatusCondition{
						{
							Code:          pb.StatusCondition_GCE_STOCKOUT,
							Message:       "test",
							CanonicalCode: 1,
						},
					},
					PodIpv4CidrSize: 100,
					UpgradeSettings: &pb.NodePool_UpgradeSettings{
						MaxSurge:       100,
						MaxUnavailable: 100,
						Strategy:       lo.ToPtr(pb.NodePoolUpdateStrategy_BLUE_GREEN),
						BlueGreenSettings: &pb.BlueGreenSettings{
							RolloutPolicy: &pb.BlueGreenSettings_StandardRolloutPolicy_{
								StandardRolloutPolicy: &pb.BlueGreenSettings_StandardRolloutPolicy{
									UpdateBatchSize: &pb.BlueGreenSettings_StandardRolloutPolicy_BatchNodeCount{
										BatchNodeCount: 10,
									},
									BatchSoakDuration: durationpb.New(time.Second),
								},
							},
							NodePoolSoakDuration: durationpb.New(time.Second),
						},
					},
					PlacementPolicy: &pb.NodePool_PlacementPolicy{
						Type:        pb.NodePool_PlacementPolicy_COMPACT,
						TpuTopology: "test",
						PolicyName:  "test",
					},
					UpdateInfo: &pb.NodePool_UpdateInfo{
						BlueGreenInfo: &pb.NodePool_UpdateInfo_BlueGreenInfo{
							Phase:                     pb.NodePool_UpdateInfo_BlueGreenInfo_CORDONING_BLUE_POOL,
							BlueInstanceGroupUrls:     sliceStr,
							GreenInstanceGroupUrls:    sliceStr,
							BluePoolDeletionStartTime: "test",
							GreenPoolVersion:          "test",
						},
					},
					Etag:               "test",
					QueuedProvisioning: &pb.NodePool_QueuedProvisioning{Enabled: true},
					BestEffortProvisioning: &pb.BestEffortProvisioning{
						Enabled:           true,
						MinProvisionNodes: 100,
					},
				},
			},
			Locations:                sliceStr,
			EnableKubernetesAlpha:    true,
			AlphaClusterFeatureGates: sliceStr,
			ResourceLabels:           mapStr,
			LabelFingerprint:         "test",
			LegacyAbac:               &pb.LegacyAbac{Enabled: true},
			NetworkPolicy: &pb.NetworkPolicy{
				Provider: pb.NetworkPolicy_CALICO,
				Enabled:  true,
			},
			IpAllocationPolicy: &pb.IPAllocationPolicy{
				UseIpAliases:               true,
				CreateSubnetwork:           true,
				SubnetworkName:             "test",
				ClusterIpv4Cidr:            "test",
				NodeIpv4Cidr:               "test",
				ServicesIpv4Cidr:           "test",
				ClusterSecondaryRangeName:  "test",
				ServicesSecondaryRangeName: "test",
				ClusterIpv4CidrBlock:       "test",
				NodeIpv4CidrBlock:          "test",
				ServicesIpv4CidrBlock:      "test",
				TpuIpv4CidrBlock:           "test",
				UseRoutes:                  true,
				StackType:                  pb.StackType_IPV4_IPV6,
				Ipv6AccessType:             pb.IPv6AccessType_EXTERNAL,
				PodCidrOverprovisionConfig: nil,
				SubnetIpv6CidrBlock:        "test",
				ServicesIpv6CidrBlock:      "test",
				AdditionalPodRangesConfig: &pb.AdditionalPodRangesConfig{
					PodRangeNames: sliceStr,
					PodRangeInfo: []*pb.RangeInfo{
						{
							RangeName:   "test",
							Utilization: 100,
						},
						{
							RangeName:   "test",
							Utilization: 100,
						},
					},
				},
				DefaultPodIpv4RangeUtilization: 100,
				AdditionalIpRangesConfigs: []*pb.AdditionalIPRangesConfig{
					{
						Subnetwork:        "test",
						PodIpv4RangeNames: sliceStr,
					},
				},
				AutoIpamConfig: &pb.AutoIpamConfig{},
			},
			MasterAuthorizedNetworksConfig: &pb.MasterAuthorizedNetworksConfig{
				Enabled: true,
				CidrBlocks: []*pb.MasterAuthorizedNetworksConfig_CidrBlock{
					{
						DisplayName: "test",
						CidrBlock:   "test",
					},
				},
				GcpPublicCidrsAccessEnabled:       lo.ToPtr(true),
				PrivateEndpointEnforcementEnabled: lo.ToPtr(true),
			},
			MaintenancePolicy: &pb.MaintenancePolicy{
				Window: &pb.MaintenanceWindow{
					Policy: &pb.MaintenanceWindow_DailyMaintenanceWindow{
						DailyMaintenanceWindow: &pb.DailyMaintenanceWindow{
							StartTime: "test",
							Duration:  "test",
						},
					},
					MaintenanceExclusions: map[string]*pb.TimeWindow{
						"a": {
							StartTime: timestamppb.Now(),
							EndTime:   timestamppb.Now(),
						},
						"b": {
							StartTime: timestamppb.Now(),
							EndTime:   timestamppb.Now(),
						},
						"c": {
							StartTime: timestamppb.Now(),
							EndTime:   timestamppb.Now(),
						},
					},
				},
				ResourceVersion: "test",
			},
			BinaryAuthorization: &pb.BinaryAuthorization{
				Enabled:        true,
				EvaluationMode: pb.BinaryAuthorization_PROJECT_SINGLETON_POLICY_ENFORCE,
			},
			Autoscaling: &pb.ClusterAutoscaling{
				EnableNodeAutoprovisioning: true,
				ResourceLimits: []*pb.ResourceLimit{
					{
						ResourceType: "test",
						Minimum:      100,
						Maximum:      100,
					},
				},
				AutoscalingProfile: pb.ClusterAutoscaling_BALANCED,
				AutoprovisioningNodePoolDefaults: &pb.AutoprovisioningNodePoolDefaults{
					OauthScopes:    sliceStr,
					ServiceAccount: "test",
					UpgradeSettings: &pb.NodePool_UpgradeSettings{
						MaxSurge:       100,
						MaxUnavailable: 100,
						Strategy:       lo.ToPtr(pb.NodePoolUpdateStrategy_SURGE),
						BlueGreenSettings: &pb.BlueGreenSettings{
							RolloutPolicy: &pb.BlueGreenSettings_StandardRolloutPolicy_{
								StandardRolloutPolicy: &pb.BlueGreenSettings_StandardRolloutPolicy{
									UpdateBatchSize: &pb.BlueGreenSettings_StandardRolloutPolicy_BatchNodeCount{
										BatchNodeCount: 10,
									},
									BatchSoakDuration: durationpb.New(time.Second),
								},
							},
							NodePoolSoakDuration: durationpb.New(time.Second),
						},
					},
					Management: &pb.NodeManagement{
						AutoUpgrade: true,
						AutoRepair:  true,
						UpgradeOptions: &pb.AutoUpgradeOptions{
							AutoUpgradeStartTime: "test",
							Description:          "test",
						},
					},
					MinCpuPlatform: "test",
					DiskSizeGb:     100,
					DiskType:       "test",
					ShieldedInstanceConfig: &pb.ShieldedInstanceConfig{
						EnableSecureBoot:          true,
						EnableIntegrityMonitoring: true,
					},
					BootDiskKmsKey:                     "test",
					ImageType:                          "test",
					InsecureKubeletReadonlyPortEnabled: lo.ToPtr(true),
				},
				AutoprovisioningLocations: sliceStr,
				DefaultComputeClassConfig: &pb.DefaultComputeClassConfig{Enabled: true},
			},
			NetworkConfig: &pb.NetworkConfig{
				Network:                   "test",
				Subnetwork:                "test",
				EnableIntraNodeVisibility: true,
				EnableL4IlbSubsetting:     true,
				DatapathProvider:          100,
				PrivateIpv6GoogleAccess:   100,
				EnableMultiNetworking:     true,
			},
			DefaultMaxPodsConstraint: &pb.MaxPodsConstraint{MaxPodsPerNode: 100},
			ResourceUsageExportConfig: &pb.ResourceUsageExportConfig{
				BigqueryDestination: &pb.ResourceUsageExportConfig_BigQueryDestination{
					DatasetId: "test",
				},
				EnableNetworkEgressMetering: true,
				ConsumptionMeteringConfig: &pb.ResourceUsageExportConfig_ConsumptionMeteringConfig{
					Enabled: true,
				},
			},
			AuthenticatorGroupsConfig: &pb.AuthenticatorGroupsConfig{
				Enabled:       true,
				SecurityGroup: "test",
			},
			PrivateClusterConfig: &pb.PrivateClusterConfig{
				EnablePrivateNodes:        true,
				EnablePrivateEndpoint:     true,
				MasterIpv4CidrBlock:       "test",
				PrivateEndpoint:           "test",
				PublicEndpoint:            "test",
				PeeringName:               "test",
				PrivateEndpointSubnetwork: "test",
			},
			VerticalPodAutoscaling: &pb.VerticalPodAutoscaling{Enabled: true},
			ReleaseChannel:         &pb.ReleaseChannel{Channel: pb.ReleaseChannel_RAPID},
			WorkloadIdentityConfig: &pb.WorkloadIdentityConfig{WorkloadPool: "test"},
			MeshCertificates:       &pb.MeshCertificates{EnableCertificates: wrapperspb.Bool(true)},
			CostManagementConfig:   &pb.CostManagementConfig{Enabled: true},
			NotificationConfig: &pb.NotificationConfig{Pubsub: &pb.NotificationConfig_PubSub{
				Enabled: true,
				Topic:   "test",
				Filter: &pb.NotificationConfig_Filter{EventType: []pb.NotificationConfig_EventType{
					pb.NotificationConfig_UPGRADE_AVAILABLE_EVENT,
					pb.NotificationConfig_UPGRADE_EVENT,
					pb.NotificationConfig_UPGRADE_INFO_EVENT,
				}},
			}},
			SelfLink:              "test",
			Zone:                  "test",
			Endpoint:              "test",
			InitialClusterVersion: "test",
			CurrentMasterVersion:  "test",
			CurrentNodeVersion:    "test",
			CreateTime:            "test",
			Status:                100,
			StatusMessage:         "test",
			NodeIpv4CidrSize:      100,
			ServicesIpv4Cidr:      "test",
			CurrentNodeCount:      100,
			ExpireTime:            "test",
			Location:              "test",
			EnableTpu:             true,
			TpuIpv4CidrBlock:      "test",
			Autopilot: &pb.Autopilot{
				Enabled: true,
				WorkloadPolicyConfig: &pb.WorkloadPolicyConfig{
					AllowNetAdmin:                         lo.ToPtr(true),
					AutopilotCompatibilityAuditingEnabled: lo.ToPtr(true),
				},
			},
			Id:   "test",
			Etag: "test",
		},
		Parent: "test",
	}
}

var (
	sliceStr = []string{"a", "b", "c"}
	mapStr   = map[string]string{"a": "1", "b": "2", "c": "3"}
)
