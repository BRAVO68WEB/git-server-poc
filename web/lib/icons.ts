import { generateManifest } from "material-icon-theme";
import { FileEntry } from "./types";

const manifest = generateManifest({
  activeIconPack: "react",
});

export function getIconSrc(entry: FileEntry): string {
  const isFolder = entry.type === "tree";
  const name = entry.name;
  let iconId: string | undefined;

  if (isFolder) {
    iconId = manifest.folderNames?.[name] || manifest.folderNames?.[name.toLowerCase()];
    if (!iconId) {
      iconId = manifest.folder;
    }
  } else {
    // Check exact filename
    iconId = manifest.fileNames?.[name] || manifest.fileNames?.[name.toLowerCase()];

    if (!iconId) {
      // Check extensions
      const parts = name.split(".");
      if (parts.length > 1) {
        // Try matching extensions from longest to shortest (e.g. .test.tsx -> test.tsx, then tsx)
        for (let i = 1; i < parts.length; i++) {
          const ext = parts.slice(i).join(".");
          if (manifest.fileExtensions?.[ext]) {
            iconId = manifest.fileExtensions[ext];
            break;
          }
        }
      }
    }

    if (!iconId) {
        // Fallback to default file icon
        iconId = manifest.file;
    }
  }

  // If we still don't have an iconId, fallback to generic
  if (!iconId) {
      return isFolder ? "/icons/folder.svg" : "/icons/file.svg";
  }

  const def = manifest.iconDefinitions?.[iconId];
  if (!def || !def.iconPath) {
      return isFolder ? "/icons/folder.svg" : "/icons/file.svg";
  }

  // iconPath is relative, e.g. "./../icons/name.svg"
  // We extracted icons to /icons/
  const fileName = def.iconPath.split("/").pop();
  return `/icons/${fileName}`;
}
