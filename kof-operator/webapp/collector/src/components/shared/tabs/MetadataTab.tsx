import { MetricRow, MetricsCard } from "@/components/shared/MetricsCard";
import { Clock, Database, Link2, Tag, Tags } from "lucide-react";
import { JSX, ReactNode } from "react";
import { ObjectMeta } from "../../../models/ObjectMeta";
import JsonViewCard from "../JsonViewCard";

export interface MetadataTabProps {
  metadata: ObjectMeta;
  children?: ReactNode;
}

const MetadataTab = ({ metadata, children }: MetadataTabProps): JSX.Element => {
  return (
    <>
      <div className="grid gap-6 md:grid-cols-1 lg:grid-cols-2">
        <BasicInfoCard metadata={metadata} />
        <TimelineCard metadata={metadata} />
      </div>

      {metadata.ownerReferences && (
        <JsonViewCard
          title="Owner References"
          icon={Link2}
          data={metadata.ownerReferences ?? {}}
        />
      )}

      {metadata.labels && (
        <JsonViewCard title="Labels" icon={Tag} data={metadata.labels ?? {}} />
      )}

      {metadata.annotations && (
        <JsonViewCard
          title="Annotations"
          icon={Tags}
          data={metadata.annotations ?? {}}
          shortenTextAfterLength={50}
        />
      )}

      {children}
    </>
  );
};

export default MetadataTab;

const BasicInfoCard = ({ metadata }: { metadata: ObjectMeta }): JSX.Element => {
  const rows: MetricRow[] = [
    { title: "Name", value: metadata.name },
    { title: "Generation", value: String(metadata.generation) },
  ];

  if (metadata.namespace) {
    rows.push({ title: "Namespace", value: metadata.namespace });
  }

  return (
    <MetricsCard rows={rows} icon={Database} title={"Basic Information"} />
  );
};

const TimelineCard = ({ metadata }: { metadata: ObjectMeta }): JSX.Element => {
  const rows: MetricRow[] = [
    { title: "Created", value: metadata.creationDate.toLocaleString() },
    {
      title: "Deletion Started",
      value: metadata.deletionDate?.toLocaleString() ?? "Not Deleted",
    },
  ];

  return <MetricsCard rows={rows} icon={Clock} title={"Timeline"} />;
};
