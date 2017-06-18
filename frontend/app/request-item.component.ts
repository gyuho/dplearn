
// Request represents TypeScript version of Request in https://github.com/gyuho/deephardway/blob/master/backend/web/web.go.
export class Request {
  public data_from_frontend: string;
  public delete_request: boolean;
  constructor(
    d: string,
    delReq: boolean,
  ) {
    this.data_from_frontend = d;
    this.delete_request = delReq;
  }
}

// Item represents TypeScript version of Item in https://github.com/gyuho/deephardway/blob/master/pkg/etcd-queue/queue.go.
export class Item {
  public bucket: string;
  public created_at: string;
  public key: string;
  public value: string;
  public progress: number;
  public canceled: boolean;
  public error: string;
  public request_id: string;
  constructor(
    bucket: string,
    key: string,
    value: string,
    progress: number,
    error: string,
    reqID: string,
  ) {
    this.bucket = bucket;
    this.created_at = "";
    this.key = key;
    this.value = value;
    this.progress = progress;
    this.canceled = false;
    this.error = error;
    this.request_id = reqID;
  }
}
