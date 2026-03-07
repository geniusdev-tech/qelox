import { useEffect } from 'react';
import { check } from '@tauri-apps/plugin-updater';
import { useOrchestratorStore } from '../store';

export function useUpdater() {
    const setUpdate = useOrchestratorStore((state) => (val: any) => {
        // @ts-ignore
        state.update = val;
    });

    useEffect(() => {
        async function checkUpdate() {
            try {
                const update = await check();
                if (update) {
                    // @ts-ignore
                    useOrchestratorStore.setState({
                        update: {
                            available: true,
                            version: update.version,
                        }
                    });
                }
            } catch (err) {
                console.error('Failed to check for updates:', err);
            }
        }

        checkUpdate();
        const interval = setInterval(checkUpdate, 1000 * 60 * 60); // Check every hour
        return () => clearInterval(interval);
    }, []);
}
