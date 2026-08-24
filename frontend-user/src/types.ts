export type FileView = {
  number: number;
  level: number;
  size: number;
  entries: number;
  min_key: string;
  max_key: string;
};

export type ImmView = {
  id: number;
  bytes: number;
  entries: number;
};

export type LevelView = {
  level: number;
  files: FileView[];
  bytes: number;
  limit: number;
};

export type LSMState = {
  profile: string;
  mem_bytes: number;
  mem_limit: number;
  mem_entries: number;
  mem_ratio: number;
  immutable: ImmView[];
  levels: LevelView[];
  last_sequence: number;
  write_stall: boolean;
  compacting: boolean;
};

export type EngineEvent = {
  type: string;
  time?: string;
  payload?: Record<string, unknown>;
};

export type Toast = {
  id: number;
  kind: "ok" | "err";
  text: string;
};
