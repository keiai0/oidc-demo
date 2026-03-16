export type SignKey = {
  kid: string;
  algorithm: string;
  status: "active" | "passive" | "expired";
  created_at: string;
  rotated_at?: string;
};
