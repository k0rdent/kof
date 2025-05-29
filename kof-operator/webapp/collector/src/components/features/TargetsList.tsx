import { JSX, useState } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";

import usePrometheusTarget from "@/providers/prometheus/PrometheusHook";
import { Badge } from "../ui/badge";
import moment from "moment";
import TargetStats from "./TargetsStats";
import { Target } from "@/models/PrometheusTarget";
import JsonView from "@uiw/react-json-view";
import { Loader } from "lucide-react";
import { Button } from "../ui/button";

const TargetList = (): JSX.Element => {
  const { filteredData, loading, fetchPrometheusTargets, error } =
    usePrometheusTarget();

  if (loading) {
    return (
      <div className="flex justify-center items-center mt-32">
        <Loader className="animate-spin"></Loader>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col justify-center items-center mt-32">
        <span className="mb-3">
          Failed to fetch prometheus targets. Click "Reload" button to try
          again.
        </span>
        <Button disabled={loading} onClick={fetchPrometheusTargets}>
          Reload
        </Button>
      </div>
    );
  }

  return (
    <>
      {filteredData?.map((cluster) => (
        <div className="flex flex-col p-6" key={cluster.name}>
          <div className="flex justify-between">
            <h1 className="flex items-center text-2xl w-fit font-bold ml-2">{`${cluster.name}`}</h1>
            <TargetStats clusters={[cluster]}></TargetStats>
          </div>
          <Table className="w-full table-fixed">
            <TableHeader>
              <TableRow>
                <TableHead className="w-[45%]">Endpoint</TableHead>
                <TableHead className="w-[7%]">State</TableHead>
                <TableHead className="w-[10%]">Labels</TableHead>
                <TableHead className="w-[13%]">Last Scrape</TableHead>
                <TableHead className="w-[10%]">Scrape Duration</TableHead>
                <TableHead className="w-[15%] text-right">Error</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody className="w-full">
              {cluster.targets.map((target, idx) => (
                <Row
                  key={`${cluster.name}-${target.scrapeUrl}-${idx}`}
                  target={target}
                />
              ))}
            </TableBody>
          </Table>
        </div>
      ))}
    </>
  );
};

export default TargetList;

const Row = ({ target }: { target: Target }) => {
  const [open, setOpen] = useState(false);
  const prettyScrapeUrl = new URL(target.scrapeUrl);
  prettyScrapeUrl.host =
    target.discoveredLabels["__meta_kubernetes_pod_name"] ||
    prettyScrapeUrl.host;

  return (
    <>
      <TableRow
        className="cursor-pointer hover:bg-muted transition-colors"
        onClick={() => setOpen((o) => !o)}
      >
        <EndpointCell url={prettyScrapeUrl.toString()} />
        <StateCell state={target.health} />
        <LabelsCell labels={target.labels} />
        <LastScrapeCell date={target.lastScrape} />
        <ScrapeDurationCell duration={target.lastScrapeDuration} />
        <ErrorCell message={target.lastError} />
      </TableRow>

      {open && (
        <TableRow className="bg-muted/40">
          <TableCell colSpan={6} className="p-4">
            <JsonView
              value={target}
              displayDataTypes={false}
              className="w-full whitespace-normal break-words"
            />
          </TableCell>
        </TableRow>
      )}
    </>
  );
};

const LastScrapeCell = ({ date }: { date: Date }): JSX.Element => {
  const m = moment(date);
  return (
    <TableCell>
      {!m.isValid() || m.year() === 1 ? "Unknown" : m.fromNow()}
    </TableCell>
  );
};

const ScrapeDurationCell = ({
  duration,
}: {
  duration: number;
}): JSX.Element => {
  return (
    <TableCell>
      {moment.duration(duration, "seconds").asMilliseconds().toFixed(3)}
      ms
    </TableCell>
  );
};

const EndpointCell = ({ url }: { url: string }): JSX.Element => {
  return (
    <TableCell className="font-medium truncate">
      <HoverCard>
        <HoverCardTrigger>{url}</HoverCardTrigger>
        <HoverCardContent className="w-fit">{url}</HoverCardContent>
      </HoverCard>
    </TableCell>
  );
};

const ErrorCell = ({
  message,
}: {
  message: string | undefined;
}): JSX.Element => {
  return (
    <TableCell className="text-right truncate">
      <HoverCard>
        <HoverCardTrigger>{message ?? ""}</HoverCardTrigger>
        <HoverCardContent className="w-fit">{message}</HoverCardContent>
      </HoverCard>
    </TableCell>
  );
};

const StateCell = ({ state }: { state: string }): JSX.Element => {
  const color =
    state === "up"
      ? "bg-green-500"
      : state === "down"
      ? "bg-red-500"
      : "bg-amber-300 text-black";
  return (
    <TableCell>
      <Badge className={`${color} border-0 capitalize`}>{state}</Badge>
    </TableCell>
  );
};

const LabelsCell = ({
  labels,
}: {
  labels: Record<string, string>;
}): JSX.Element => {
  const count = Object.keys(labels).length;
  return (
    <TableCell className="max-w-64">
      <div className="flex items-center gap-1">
        <span>{count} Labels</span>
      </div>
    </TableCell>
  );
};
