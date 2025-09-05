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
import { useClusterSummariesProvider } from "@/providers/ClusterSummariesProvider";
import MetadataTab from "@/components/shared/tabs/MetadataTab";
import RawJsonTab from "@/components/shared/tabs/RawJsonTab";
import StatusTab from "@/components/shared/tabs/StatusTab";
import DetailsHeader from "@/components/shared/DetailsHeader";

const ClusterSummaryDetails = (): JSX.Element => {
  const {
    isLoading,
    items: summaries,
    selectedItem: summary,
    selectItem,
  } = useClusterSummariesProvider();

  const navigate = useNavigate();
  const { summaryName } = useParams();

  useEffect(() => {
    if (!isLoading && summaries && summaryName) {
      selectItem(summaryName);
    }
  }, [summaryName, summaries, isLoading, selectItem]);

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
      <DetailsHeader
        backPath={"/cluster-summaries"}
        icon={Layers2}
        title={summary.name}
        isHealthy={summary.isHealthy}
      >
        <Button
          variant="outline"
          className="cursor-pointer"
          onClick={() => {
            navigate(`/cluster-deployments/${summary.spec.clusterName}`);
          }}
        >
          <span>Go to Cluster Deployment</span>
          <MoveRight />
        </Button>
      </DetailsHeader>
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
        <StatusTab conditions={summary.status.featureSummaries.arr} />
        <MetadataTab metadata={summary.metadata} />
        <RawJsonTab depthLevel={4} object={summary.raw} />
        <StatusTab
          conditions={summary.status.helmReleaseSummaries.arr}
          tabValue="helm"
        />
      </Tabs>
    </div>
  );
};

export default ClusterSummaryDetails;

const UnhealthyAlert = (): JSX.Element => {
  const { selectedItem: summary } = useClusterSummariesProvider();

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
