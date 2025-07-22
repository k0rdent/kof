import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/generated/ui/hover-card";
import { TableCell } from "@/components/generated/ui/table";
import { JSX } from "react";

const CollectorNameCell = ({ name }: { name: string }): JSX.Element => {
  return (
    <TableCell className="truncate">
      <HoverCard>
        <HoverCardTrigger>{name}</HoverCardTrigger>
        <HoverCardContent className="w-fit">{name}</HoverCardContent>
      </HoverCard>
    </TableCell>
  );
};

export default CollectorNameCell;
