import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/generated/ui/alert";
import { Button } from "@/components/generated/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/generated/ui/tabs";
import { AlertCircleIcon, Layers2, MoveLeft, MoveRight } from "lucide-react";
import { JSX, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useClusterSummariesProvider } from "@/providers/cluster_summaries/ClusterDeploymentsProvider";
import HealthBadge from "@/components/shared/HealthBadge";
import ClusterSummaryStatusTab from "./ClusterSummaryStatusTab";
import ClusterSummaryHelmTab from "./ClusterSummaryHelmTab";
import MetadataTab from "@/components/shared/tabs/MetadataTab";
import RawJsonTab from "@/components/shared/tabs/RawJsonTab";

const ClusterSummaryDetails = (): JSX.Element => {
  const {
    data: summaries,
    setSelectedSummary,
    selectedSummary: summary,
    isLoading,
  } = useClusterSummariesProvider();

  const navigate = useNavigate();
  const { summaryName } = useParams();

  useEffect(() => {
    if (!isLoading && summaries && summaryName) {
      setSelectedSummary(summaryName);
    }
  }, [summaryName, summaries, isLoading, setSelectedSummary]);

  if (!summary) {
    return (
      <div className="flex flex-col w-full h-full p-5 space-y-8">
        <div className="flex flex-col w-full h-full justify-center items-center space-y-4">
          <span className="font-bold text-2xl">Cluster summary not found</span>
          <Button
            variant="outline"
            className="cursor-pointer"
            onClick={() => {
              navigate("/cluster-summaries");
            }}
          >
            <MoveLeft />
            <span>Back to Table</span>
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <DetailsHeader />
      <UnhealthyAlert />
      <Tabs defaultValue="status" className="space-y-6">
        <TabsList className="flex w-full">
          <TabsTrigger value="status">Status</TabsTrigger>
          {(summary.status.helmReleaseSummaries.count ?? 0) > 0 && (
            <TabsTrigger value="helm">Helm Charts</TabsTrigger>
          )}
          <TabsTrigger value="metadata">Metadata</TabsTrigger>
          <TabsTrigger value="raw_json">Raw Json</TabsTrigger>
        </TabsList>
        <ClusterSummaryStatusTab />
        <MetadataTab metadata={summary.metadata} />
        <ClusterSummaryHelmTab />
        <RawJsonTab depthLevel={4} object={summary.rawData} />
      </Tabs>
    </div>
  );
};

export default ClusterSummaryDetails;

const UnhealthyAlert = (): JSX.Element => {
  const { selectedSummary: summary } = useClusterSummariesProvider();

  if (!summary || summary.isHealthy || !summary.status.failureMessage)
    return <></>;

  const messages = Array.isArray(summary.status.failureMessage)
    ? summary.status.failureMessage
    : [summary.status.failureMessage];

  return (
    <Alert variant="destructive">
      <AlertCircleIcon className="w-6 h-6 text-red-600" />
      <AlertTitle className="mb-2">Cluster Summary Errors:</AlertTitle>
      <AlertDescription className="flex flex-col space-y-3">
        <ul className="list-disc list-inside space-y-2">
          {messages.map((msg, idx) => (
            <li key={idx} className="border-l-2 border-red-500 pl-3">
              <span>{msg}</span>
            </li>
          ))}
        </ul>
      </AlertDescription>
    </Alert>
  );
};

const DetailsHeader = (): JSX.Element => {
  const { selectedSummary: summary } = useClusterSummariesProvider();
  const navigate = useNavigate();

  return (
    <div className="flex flex-col space-y-6">
      <div className="flex items-center space-x-6">
        <Button
          variant="outline"
          className="cursor-pointer"
          onClick={() => {
            navigate("/cluster-summaries");
          }}
        >
          <MoveLeft />
          <span>Back to Table</span>
        </Button>

        <Button
          variant="outline"
          className="cursor-pointer"
          onClick={() => {
            navigate(`/cluster-deployments/${summary?.spec.clusterName}`);
          }}
        >
          <span>Go to Cluster Deployment</span>
          <MoveRight />
        </Button>
      </div>
      <div className="flex gap-4 items-center mb-2">
        <Layers2 />
        <span className="font-bold text-xl">{summary?.name}</span>
        <HealthBadge isHealthy={summary?.isHealthy ?? false} />
      </div>
    </div>
  );
};
