import { JSX, useEffect, useRef, useState } from "react";
import { Label } from "../ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "../ui/popover";
import { Button } from "../ui/button";
import { Check, ChevronsUpDown } from "lucide-react";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "../ui/command";
import { cn } from "@/lib/utils";
import usePrometheusTarget from "@/providers/prometheus/PrometheusHook";

import { FilterFunction } from "@/providers/prometheus/PrometheusTargetsProvider";
import { Cluster } from "@/models/Cluster";
import { Node } from "@/models/Node";

export const PopoverClusterFilter = (
  selectedValues: string[]
): FilterFunction => {
  return (data: Cluster[]) => {
    return data.filter((cluster) => selectedValues.includes(cluster.name));
  };
};

export const PopoverNodeFilter = (selectedValues: string[]): FilterFunction => {
  return (data: Cluster[]) => {
    return data
      .map((cluster) => {
        return new Cluster(cluster.name, cluster.findNodes(selectedValues));
      })
      .filter((cluster) => cluster.hasNodes);
  };
};

interface PopoverSelectorProps {
  id: string;
  labelContent: string;
  noValuesText: string;
  placeholderText: string;
  popoverButtonText: string;
  dataToDisplay: Cluster[] | Node[];
  filterFn: (selectedValues: string[]) => FilterFunction;
  onSelectionChange?: (selectedValues: string[]) => void;
}

const PopoverSelector = ({
  id,
  labelContent,
  noValuesText,
  popoverButtonText,
  placeholderText,
  dataToDisplay,
  filterFn,
  onSelectionChange,
}: PopoverSelectorProps): JSX.Element => {
  const [openPopover, setPopoverOpen] = useState(false);
  const [selectedValues, setSelectedValues] = useState<string[]>([]);
  const { loading, addFilter, removeFilter } = usePrometheusTarget();

  const filterIdRef = useRef<string | null>(null);

  const handleSelect = (name: string) => {
    setSelectedValues((prevSelected) => {
      if (prevSelected.includes(name)) {
        return prevSelected.filter((item) => item !== name);
      } else {
        return [...prevSelected, name];
      }
    });
  };

  useEffect(() => {
    if (filterIdRef.current) {
      removeFilter(filterIdRef.current);
    }

    if (selectedValues.length > 0) {
      const newFilterId = addFilter(
        `${id}_popover_filter`,
        filterFn(selectedValues)
      );
      filterIdRef.current = newFilterId;
    } else {
      filterIdRef.current = null;
    }

    if (onSelectionChange) {
      onSelectionChange(selectedValues);
    }
  }, [
    selectedValues,
    id,
    addFilter,
    filterFn,
    onSelectionChange,
    removeFilter,
  ]);

  return (
    <div className="flex gap-2">
      <Label className="font-bold text-lg">{labelContent}</Label>
      <Popover open={openPopover} onOpenChange={setPopoverOpen}>
        <PopoverTrigger className="cursor-pointer" disabled={loading} asChild>
          <Button
            variant="outline"
            role="combobox"
            aria-expanded={openPopover}
            className="w-[250px] justify-between"
          >
            {selectedValues.length > 0
              ? `${selectedValues.length} selected`
              : popoverButtonText}
            <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-[250px] p-0">
          <Command>
            <CommandInput placeholder={placeholderText} />
            <CommandList>
              <CommandEmpty>{noValuesText}</CommandEmpty>
              <CommandGroup>
                {dataToDisplay?.map((data) => (
                  <CommandItem
                    key={data.name}
                    value={data.name}
                    onSelect={() => handleSelect(data.name)}
                  >
                    <Check
                      className={cn(
                        "mr-2 h-4 w-4",
                        selectedValues.includes(data.name)
                          ? "opacity-100"
                          : "opacity-0"
                      )}
                    />
                    {data.name}
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>
  );
};

export default PopoverSelector;
