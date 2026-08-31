import { redirect } from "next/navigation";

export default function NotifierSlackRedirect() {
  redirect("/notifiers/slack");
}
