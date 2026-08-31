import { FileManager } from "@/components/files/FileManager";
import { getMe } from "@/lib/auth-server";

export default async function FilesPage() {
  const me = await getMe();
  return (
    <FileManager
      workspace={me?.workspace ? { id: me.workspace.id, name: me.workspace.name } : null}
    />
  );
}
