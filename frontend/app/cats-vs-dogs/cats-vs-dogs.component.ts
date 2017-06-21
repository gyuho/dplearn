import {
  Component,
} from "@angular/core";

import {
  BackendService,
} from "../request.service";

@Component({
  providers: [BackendService],
  selector: "app",
  styleUrls: ["cats-vs-dogs.component.css"],
  templateUrl: "cats-vs-dogs.component.html",
})
export class CatsVsDogsComponent {
  public endpoint = "cats-vs-dogs-request";
  constructor(public backendService: BackendService) {
    backendService.endpoint = this.endpoint;
    backendService.inputValue = "https://images.pexels.com/photos/127028/pexels-photo-127028.jpeg?w=1260&h=750&auto=compress&cs=tinysrgb";
  }
}
