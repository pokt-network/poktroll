// @ts-check
// `@type` JSDoc annotations allow editor autocompletion and type checking
// (when paired with `@ts-check`).
// There are various equivalent ways to declare your Docusaurus config.
// See: https://docusaurus.io/docs/api/docusaurus-config

import { themes as prismThemes } from "prism-react-renderer";

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: "Poktroll",
  tagline: "Roll the POKT",
  favicon: "img/logo.png",

  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'

  // GitHub pages deployment config.
  url: "https://poktroll.com/",
  baseUrl: "/",
  // Custom domain config.
  // url: "https://docs.poktroll.com",
  // baseUrl: "/",

  markdown: { mermaid: true },
  themes: ["@docusaurus/theme-mermaid"],

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: "pokt-network", // Usually your GitHub org/user name (ORGANIZATION_NAME)
  projectName: "poktroll", // Usually your repo name. (PROJECT_NAME)
  deploymentBranch: "gh-pages", // Deployment branch (DEPLOYMENT_BRANCH)
  trailingSlash: false,

  onBrokenLinks: "throw",
  onBrokenMarkdownLinks: "warn",

  // Even if you don't use internationalization, you can use this field to set
  // useful metadata like html lang. For example, if your site is Chinese, you
  // may want to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },

  presets: [
    [
      // "classic",
      "@docusaurus/preset-classic",
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          routeBasePath: "/",
          sidebarPath: "./sidebars.js",
        },
        theme: {},
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      // Replace with your project's social card
      // image: "img/docusaurus-social-card.jpg",
      style: "dark",
      navbar: {
        title: "Pocket Network",
        logo: {
          alt: "Pocket Network Logo",
          src: "img/logo.png",
        },
        items: [],
      },
      footer: {
        style: "dark",
        links: [
          {
            title: "Documentation",
            items: [
              {
                label: "Poktroll rollup",
                to: "/",
              },
              {
                label: "Pocket Network",
                href: "https://docs.pokt.network/",
              },
            ],
          },
          {
            title: "Community",
            items: [
              {
                label: "Discord - Pocket",
                href: "https://discord.gg/6cKbYA2X",
              },
              {
                label: "Twitter",
                href: "https://twitter.com/poktnetwork",
              },
            ],
          },
          {
            title: "More",
            items: [
              {
                label: "GitHub",
                href: "https://github.com/pokt-network/poktroll",
              },
            ],
          },
        ],
        copyright: `MIT License © Pocket Network`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
      },
    }),
};

export default config;
