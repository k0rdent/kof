import { JSX, useEffect, useMemo } from "react";
import { Target } from "@/models/PrometheusTarget";
import { Cluster } from "@/models/Cluster";
import { Alert, AlertDescription, AlertTitle } from "../generated/ui/alert";
import { AlertCircleIcon } from "lucide-react";
import { toast } from "sonner";
import URL from "url";
import usePrometheusTarget from "@/providers/prometheus/PrometheusHook";

const DuplicateTargetsAlert = ({
  clusterName,
}: {
  clusterName: string;
}): JSX.Element => {
  const { data } = usePrometheusTarget();

  const cluster: Cluster | undefined = useMemo(() => {
    return data?.findCluster(clusterName);
  }, [data, clusterName]);

  const targetsMap = useMemo(() => {
    const map: Record<string, Target[]> = {};
    if (!cluster || !cluster.targets) return map;

    cluster.targets.forEach((target) => {
      map[target.scrapeUrl] = map[target.scrapeUrl] ?? [];
      map[target.scrapeUrl].push(target);
    });
    return map;
  }, [cluster]);

  const duplicatedScrapeUrl = useMemo(
    () =>
      Object.entries(targetsMap)
        .filter(([scrapeUrl, targets]) => targets.length > 1 && (
          !['127.0.0.1', 'localhost'].includes(URL.parse(scrapeUrl, true)?.hostname ?? '') ||
          targets.some(t => t.node !== targets[0].node)
        ))
        .map(([key]) => key),
    [targetsMap]
  );

  useEffect(() => {
    if (cluster && duplicatedScrapeUrl.length !== 0) {
      toast.error("Misconfiguration Found!", {
        id: cluster.name,
        description: `Check the '${cluster.name}' cluster for more details.`,
      });
    }
  }, [cluster, duplicatedScrapeUrl]);

  if (!cluster || duplicatedScrapeUrl.length === 0) return <></>;

  return (
    <Alert variant="destructive">
      <AlertCircleIcon />
      <AlertTitle>Misconfiguration Found!</AlertTitle>
      <AlertDescription className="flex flex-col space-y-2">
        <span>Some targets are duplicated and scraping the same URL</span>
        {duplicatedScrapeUrl.map((scrapeUrl, index) => {
          return (
            <div key={`${scrapeUrl}-${index}`} className="border-l-2 border-red-500 pl-3">
              <p className="font-semibold">Scrape URL: {scrapeUrl}</p>
              <ul className="list-disc list-inside">
                {targetsMap[scrapeUrl].map((target, index) => (
                  <li key={`${cluster.name}-${target.scrapePool}-${index}`}>
                    <button
                      className="cursor-pointer"
                      onClick={() => {
                        const id: string = `${cluster.name}-${target.scrapePool}-${target.lastScrape}`;
                        const el = document.getElementById(id);
                        if (!el) return;
                        highlightElement(el);
                      }}
                    >
                      <span>node: {target.node}, pool {target.scrapePool}</span>
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          );
        })}
      </AlertDescription>
    </Alert>
  );
};

const highlightElement = (el: HTMLElement) => {
  el.scrollIntoView({ behavior: "smooth", block: "center" });
  el.focus({ preventScroll: true });
  el.classList.add("transition-colors", "duration-500", "bg-red-100");
  setTimeout(() => el.classList.remove("bg-red-100"), 2000);
};

export default DuplicateTargetsAlert;
