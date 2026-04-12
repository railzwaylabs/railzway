import { NavLink } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useOrgIdParam, useOrgPath } from "../lib/org";

export default function NotFound() {
  const { t } = useTranslation();
  const orgId = useOrgIdParam();
  const orgPath = useOrgPath();
  const backTo = orgId ? orgPath("/dashboard") : "/organizations";
  return (
    <div className="page">
      <section className="page-hero">
        <h2>{t("not_found.title")}</h2>
        <p>{t("not_found.description")}</p>
        <NavLink className="nav-link active" to={backTo}>
          <span className="nav-dot" />
          {t("not_found.back")}
        </NavLink>
      </section>
    </div>
  );
}
