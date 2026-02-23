export type SignKey = {
  kid: string;
  algorithm: string;
  active: boolean;
  created_at: string;
  rotated_at?: string;
};
