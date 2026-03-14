import { LinkObject, NodeObject } from "react-force-graph-2d";

export interface GraphNode extends NodeObject {
  id: string;
  name: string;
  namespace: string;
  role: string;
}

export interface GraphLink extends LinkObject {
  secretName: string;
}
