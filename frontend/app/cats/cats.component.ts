import {
  Component,
} from "@angular/core";

import {
  BackendService,
} from "../request.service";

@Component({
  providers: [BackendService],
  selector: "app",
  styleUrls: ["cats.component.css"],
  templateUrl: "cats.component.html",
})
export class CatsComponent {
  public endpoint = "cats-request";
  constructor(public backendService: BackendService) {
    backendService.endpoint = this.endpoint;
    backendService.inputValue = "https://static.pexels.com/photos/127028/pexels-photo-127028.jpeg";
  }
}
