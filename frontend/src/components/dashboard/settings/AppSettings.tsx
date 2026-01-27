import { Input, Label, Select, ListBox } from "@heroui/react";
import React from "react";
import { useSettingsStore } from "../../../store/useSettingsStore";

export const AppSettings: React.FC = () => {
  const { serverSettings, updateServerSettings } = useSettingsStore();

  if (!serverSettings || !serverSettings.download) return null;

  const setDefaultDownloadDir = (val: string) => {
    updateServerSettings({
      download: { 
        ...serverSettings.download!, 
        downloadDir: val,
        preferredEngine: serverSettings.download?.preferredEngine || "aria2",
        preferredMagnetEngine: serverSettings.download?.preferredMagnetEngine || "aria2",
      }
    });
  };

  const setPreferredEngine = (val: "aria2" | "native") => {
    updateServerSettings({
      download: { 
        ...serverSettings.download!, 
        preferredEngine: val,
        downloadDir: serverSettings.download?.downloadDir || "/downloads",
        preferredMagnetEngine: serverSettings.download?.preferredMagnetEngine || "aria2",
      }
    });
  };

  return (
    <div className="space-y-10">
      <section className="space-y-6">
        <div className="border-b border-border pb-2">
          <h3 className="text-lg font-bold">Storage</h3>
          <p className="text-sm text-muted">
            Configure where your downloads are saved by default.
          </p>
        </div>

        <div className="flex flex-col gap-2">
          <Label className="text-sm font-bold tracking-tight">
            Default Download Directory
          </Label>
          <Input
            value={serverSettings.download.downloadDir}
            onChange={(e) => setDefaultDownloadDir(e.target.value)}
            placeholder="e.g. /downloads"
            className="w-full bg-default/10 rounded-xl"
          />
        </div>
      </section>

      <section className="space-y-6">
        <div className="border-b border-border pb-2">
          <h3 className="text-lg font-bold">Engine</h3>
          <p className="text-sm text-muted">
            Choose which engine to use for new downloads.
          </p>
        </div>

        <div className="flex flex-col gap-2">
          <Label className="text-sm font-bold tracking-tight">
            Preferred Download Engine
          </Label>
          <Select
            selectedKey={serverSettings.download.preferredEngine}
            onSelectionChange={(key) => setPreferredEngine(key as "aria2" | "native")}
            className="w-full bg-default/10 rounded-xl"
          >
            <ListBox>
              <ListBox.Item id="aria2" textValue="aria2c">aria2c (External)</ListBox.Item>
              <ListBox.Item id="native" textValue="Native">Native (Go-based)</ListBox.Item>
            </ListBox>
          </Select>
        </div>
      </section>
    </div>
  );
};
