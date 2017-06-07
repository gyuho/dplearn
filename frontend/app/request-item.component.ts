
// Request represents TypeScript version of Request in https://github.com/gyuho/deephardway/blob/master/backend/web/web.go.
export class Request {
  userid: string;
  url: string;
  rawdata: string;
  result: string;
  constructor(
    url: string,
    rawdata: string,
  ) {
    this.userid = '';
    this.url = url;
    this.rawdata = rawdata;
    this.result = '';
  }
};

// Item represents TypeScript version of Item in https://github.com/gyuho/deephardway/blob/master/pkg/etcd-queue/queue.go.
export class Item {
  bucket: string;
  key: string;
  value: string;
  progress: number;
  error: string;
  constructor(
    bucket: string,
    key: string,
    value: string,
    progress: number,
    error: string,
  ) {
    this.bucket = bucket;
    this.key = key;
    this.value = value;
    this.progress = progress;
    this.error = error;
  }
};
