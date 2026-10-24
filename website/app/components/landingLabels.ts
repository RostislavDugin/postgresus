import type { Locale } from "@/app/i18n";

export interface LandingLabels {
  logoAlt: string;
  docs: string;
  community: string;
  starOnGithub: string;
  postgresqlBackup: string;
  mysqlBackup: string;
  mongodbBackup: string;
  documentation: string;
  developer: string;
  privacy: string;
  termsOfUse: string;
  copyright: string;
}

export const LANDING_LABELS: Record<Locale | "en", LandingLabels> = {
  en: {
    logoAlt: "Databasus logo",
    docs: "Docs",
    community: "Community",
    starOnGithub: "Star on GitHub, it's really important ❤️",
    postgresqlBackup: "PostgreSQL backup",
    mysqlBackup: "MySQL and MariaDB backup",
    mongodbBackup: "MongoDB backup",
    documentation: "Documentation",
    developer: "Developer",
    privacy: "Privacy",
    termsOfUse: "Terms of use",
    copyright: "© 2026 Databasus™. All rights reserved.",
  },
  ru: {
    logoAlt: "Логотип Databasus",
    docs: "Документация",
    community: "Сообщество",
    starOnGithub: "Поддержать проект звездой  ❤️",
    postgresqlBackup: "Резервное копирование PostgreSQL",
    mysqlBackup: "Резервное копирование MySQL и MariaDB",
    mongodbBackup: "Резервное копирование MongoDB",
    documentation: "Документация",
    developer: "Разработчик",
    privacy: "Конфиденциальность",
    termsOfUse: "Условия использования",
    copyright: "© 2026 Databasus™. Все права защищены.",
  },
  es: {
    logoAlt: "Logotipo de Databasus",
    docs: "Documentación",
    community: "Comunidad",
    starOnGithub: "Deje una estrella en GitHub ❤️",
    postgresqlBackup: "Copia de seguridad de PostgreSQL",
    mysqlBackup: "Copia de seguridad de MySQL y MariaDB",
    mongodbBackup: "Copia de seguridad de MongoDB",
    documentation: "Documentación",
    developer: "Desarrollador",
    privacy: "Privacidad",
    termsOfUse: "Términos de uso",
    copyright: "© 2026 Databasus™. Todos los derechos reservados.",
  },
  pt: {
    logoAlt: "Logotipo do Databasus",
    docs: "Docs",
    community: "Comunidade",
    starOnGithub: "Dê uma estrela no GitHub ❤️",
    postgresqlBackup: "Backup PostgreSQL",
    mysqlBackup: "Backup MySQL e MariaDB",
    mongodbBackup: "Backup MongoDB",
    documentation: "Documentação",
    developer: "Desenvolvedor",
    privacy: "Privacidade",
    termsOfUse: "Termos de uso",
    copyright: "© 2026 Databasus™. Todos os direitos reservados.",
  },
  zh: {
    logoAlt: "Databasus logo",
    docs: "文档",
    community: "社区",
    starOnGithub: "在 GitHub 上点个 Star ❤️",
    postgresqlBackup: "PostgreSQL 备份",
    mysqlBackup: "MySQL 和 MariaDB 备份",
    mongodbBackup: "MongoDB 备份",
    documentation: "文档",
    developer: "开发者",
    privacy: "隐私政策",
    termsOfUse: "使用条款",
    copyright: "© 2026 Databasus™。保留所有权利。",
  },
  fr: {
    logoAlt: "Logo Databasus",
    docs: "Docs",
    community: "Communauté",
    starOnGithub: "Mettez une étoile sur GitHub ❤️",
    postgresqlBackup: "Sauvegarde PostgreSQL",
    mysqlBackup: "Sauvegarde MySQL et MariaDB",
    mongodbBackup: "Sauvegarde MongoDB",
    documentation: "Documentation",
    developer: "Développeur",
    privacy: "Confidentialité",
    termsOfUse: "Conditions d'utilisation",
    copyright: "© 2026 Databasus™. Tous droits réservés.",
  },
};
