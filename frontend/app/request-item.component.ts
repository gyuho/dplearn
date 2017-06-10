
// Request represents TypeScript version of Request in https://github.com/gyuho/deephardway/blob/master/backend/web/web.go.
export class Request {
  user_id: string;
  url: string;
  raw_data: string;
  result: string;
  constructor(
    url: string,
    d: string,
  ) {
    this.user_id = '';
    this.url = url;
    this.raw_data = d;
    this.result = '';
  }
};

// Item represents TypeScript version of Item in https://github.com/gyuho/deephardway/blob/master/pkg/etcd-queue/queue.go.
export class Item {
  bucket: string;
  created_at: string;
  key: string;
  value: string;
  progress: number;
  canceled: boolean;
  error: string;
  constructor(
    bucket: string,
    key: string,
    value: string,
    progress: number,
    error: string,
  ) {
    this.bucket = bucket;
    this.created_at = '';
    this.key = key;
    this.value = value;
    this.progress = progress;
    this.canceled = false;
    this.error = error;
  }
};
