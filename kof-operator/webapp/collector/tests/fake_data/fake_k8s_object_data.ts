import { Settings } from "lucide-react";
import {
  DetailsTab,
  TableColumn,
} from "../../src/components/pages/dashboards/DashboardTypes";
import {
  TableColAge,
  TableColName,
  TableColStatus,
} from "../../src/components/pages/dashboards/TableColumns";
import { DefaultCondition, DefaultStatus } from "../../src/models/DefaultCondition";
import { K8sObject, K8sObjectData } from "../../src/models/k8sObject";
import { K8sObjectSet } from "../../src/models/k8sObjectSet";
import { ClusterConditionData, Condition } from "../../src/models/ObjectMeta";
import {
  DetailMetadataTab,
  DetailRawJsonTab,
  DetailStatusTab,
} from "../../src/components/pages/dashboards/DetailTabs";
import { vi } from "vitest";

export class FakeK8sObjectSet extends K8sObjectSet<FakeK8sObject> {
  protected createK8sObject(
    path: string,
    data: K8sObjectData<FakeSpecData, FakeStatusData>,
  ): FakeK8sObject {
    return new FakeK8sObject(path, data);
  }
}

export class FakeK8sObject extends K8sObject<
  FakeSpec,
  FakeStatus,
  FakeSpecData,
  FakeStatusData
> {
  public get isHealthy(): boolean {
    return this.status.conditions.find((c: Condition) => !c.isHealthy) ? false : true;
  }

  protected createSpec(raw: FakeSpecData): FakeSpec {
    return new FakeSpec(raw);
  }

  protected createStatus(raw: FakeStatusData): FakeStatus {
    return new FakeStatus(raw);
  }
}

export class FakeSpec {
  constructor(public data: FakeSpecData) {}
}

export class FakeStatus implements DefaultStatus {
  conditions: DefaultCondition[] = [];
  constructor(public data: FakeStatusData) {
    this.data.conditions.forEach((c) => this.conditions.push(new DefaultCondition(c)));
  }
}

export interface FakeSpecData {
  name: string;
  replicas: number;
}

export interface FakeStatusData {
  conditions: ClusterConditionData[];
}

export const MockK8sObjects: FakeK8sObject[] = [
  new FakeK8sObject("cluster-1/ns1/dashboard-1", {
    metadata: {
      uid: "1",
      name: "dashboard-1",
      namespace: "ns1",
      generation: 1,
      labels: {},
      annotations: {},
      creationTimestamp: new Date("2024-01-01T00:00:00Z"),
    },
    spec: { name: "dashboard-1", replicas: 1 },
    status: {
      conditions: [
        {
          type: "Ready",
          status: "True",
          reason: "Healthy",
          message: "All good",
          lastTransitionTime: "",
        },
      ],
    },
  }),
  new FakeK8sObject("cluster-1/ns2/dashboard-2",{
    metadata: {
      uid: "2",
      name: "dashboard-2",
      namespace: "ns2",
      generation: 1,
      labels: {},
      annotations: {},
      creationTimestamp: new Date("2024-01-01T00:00:00Z"),
    },
    spec: { name: "dashboard-2", replicas: 2 },
    status: {
      conditions: [
        {
          type: "Ready",
          status: "False",
          reason: "Unhealthy",
          message: "Something wrong",
          lastTransitionTime: "",
        },
      ],
    },
  }),
];

export const basicTabs = [DetailStatusTab(), DetailMetadataTab(), DetailRawJsonTab()];

export const FakeDashboardData = (
  overrides?: Partial<{
    name: string;
    id: string;
    store: () => {
      isLoading: boolean;
      items: FakeK8sObjectSet | null;
      selectedItem: FakeK8sObject | null;
      selectItem: (itemKey: string) => FakeK8sObject | undefined;
      error: Error | null;
      fetch: () => void;
    };
    tableCols: TableColumn<FakeK8sObject>[];
    detailTabs: DetailsTab<FakeK8sObject>[];
  }>,
) => {
  const defaultData = {
    name: "Test Dashboard",
    id: "test-dashboard",
    store: () => ({
      isLoading: false,
      items: mockItemSet,
      selectedItem: MockK8sObjects[0],
      selectItem: (itemKey: string) => {
        return MockK8sObjects.find((obj) => obj.name === itemKey);
      },
      fetch: vi.fn(),
    }),
    icon: Settings,
    tableCols: [TableColName(), TableColStatus(), TableColAge()],
    detailTabs: basicTabs,
  };
  return { ...defaultData, ...overrides };
};

const mockItemSet = {
  objects: MockK8sObjects,
  length: MockK8sObjects.length,
  healthyCount: MockK8sObjects.filter((obj) => obj.isHealthy).length,
  unhealthyCount: MockK8sObjects.filter((obj) => !obj.isHealthy).length,
};
