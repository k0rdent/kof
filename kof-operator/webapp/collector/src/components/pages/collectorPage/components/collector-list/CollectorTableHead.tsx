import { TableHead } from "@/components/generated/ui/table";
import { JSX } from "react";

export interface CustomizedTableHeadProps {
  text: string;
  width: number;
}

const CustomizedTableHead = ({
  text,
  width,
}: CustomizedTableHeadProps): JSX.Element => {
  return (
    <TableHead
      className="text-left text-base font-semibold"
      style={{ width: `${width}px` }}
    >
      {text}
    </TableHead>
  );
};

export default CustomizedTableHead;
