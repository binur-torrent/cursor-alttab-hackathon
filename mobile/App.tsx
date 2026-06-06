import { useEffect, useState } from "react";
import {
  ActivityIndicator,
  FlatList,
  Modal,
  Pressable,
  SafeAreaView,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { StatusBar } from "expo-status-bar";
import MapView, { Marker } from "react-native-maps";
import * as Location from "expo-location";

import { api } from "./src/api";
import { colors, riskColor } from "./src/theme";
import type { AnalyzeResult, SegmentDetail, StreetSegment } from "./src/types";

type Tab = "nearby" | "report";

export default function App() {
  const [tab, setTab] = useState<Tab>("nearby");

  return (
    <SafeAreaView style={styles.root}>
      <StatusBar style="light" />
      <View style={styles.header}>
        <View style={styles.dot} />
        <View>
          <Text style={styles.title}>LumiCity AI</Text>
          <Text style={styles.subtitle}>Istanbul streetlight field tool</Text>
        </View>
      </View>

      <View style={styles.tabs}>
        {(["nearby", "report"] as Tab[]).map((t) => (
          <Pressable
            key={t}
            onPress={() => setTab(t)}
            style={[styles.tab, tab === t && styles.tabActive]}
          >
            <Text style={[styles.tabText, tab === t && styles.tabTextActive]}>
              {t === "nearby" ? "Nearby risks" : "Report a dark spot"}
            </Text>
          </Pressable>
        ))}
      </View>

      {tab === "nearby" ? <NearbyScreen /> : <ReportScreen />}
    </SafeAreaView>
  );
}

function NearbyScreen() {
  const [segments, setSegments] = useState<StreetSegment[]>([]);
  const [loading, setLoading] = useState(true);
  const [detail, setDetail] = useState<SegmentDetail | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .problemSegments()
      .then((r) => setSegments(r.data))
      .catch((e) => setError(String(e)))
      .finally(() => setLoading(false));
  }, []);

  async function openSegment(s: StreetSegment) {
    try {
      const d = await api.segment(s.external_id);
      setDetail(d);
    } catch (e) {
      setError(String(e));
    }
  }

  if (loading) return <Centered><ActivityIndicator color={colors.accent} /></Centered>;
  if (error) return <Centered><Text style={styles.error}>{error}</Text></Centered>;

  return (
    <View style={{ flex: 1 }}>
      <MapView
        style={styles.map}
        initialRegion={{
          latitude: 41.02,
          longitude: 29.0,
          latitudeDelta: 0.35,
          longitudeDelta: 0.35,
        }}
      >
        {segments.map((s) => (
          <Marker
            key={s.external_id}
            coordinate={{ latitude: s.centroid_lat, longitude: s.centroid_lon }}
            pinColor={riskColor(s.risk_level)}
            title={s.name}
            description={`${s.district} · risk ${s.risk_score} · ${s.street_light_count} lights`}
            onCalloutPress={() => openSegment(s)}
          />
        ))}
      </MapView>

      <FlatList
        style={styles.list}
        data={segments}
        keyExtractor={(s) => s.external_id}
        ListHeaderComponent={
          <Text style={styles.listHeader}>{segments.length} high-risk segments</Text>
        }
        renderItem={({ item }) => (
          <Pressable style={styles.row} onPress={() => openSegment(item)}>
            <View style={[styles.riskChip, { backgroundColor: riskColor(item.risk_level) }]} />
            <View style={{ flex: 1 }}>
              <Text style={styles.rowTitle} numberOfLines={1}>{item.name}</Text>
              <Text style={styles.rowSub}>
                {item.district} · {item.street_light_count} lights · {Math.round(item.length_m)} m
              </Text>
            </View>
            <Text style={[styles.rowScore, { color: riskColor(item.risk_level) }]}>
              {item.risk_score}
            </Text>
          </Pressable>
        )}
      />

      <DetailModal detail={detail} onClose={() => setDetail(null)} />
    </View>
  );
}

function DetailModal({
  detail,
  onClose,
}: {
  detail: SegmentDetail | null;
  onClose: () => void;
}) {
  if (!detail) return null;
  const s = detail.segment;
  const blurred = detail.analyses.reduce((a, x) => a + x.faces_blurred + x.plates_blurred, 0);
  return (
    <Modal visible animationType="slide" transparent onRequestClose={onClose}>
      <View style={styles.modalWrap}>
        <View style={styles.modalCard}>
          <View style={styles.modalHandle} />
          <ScrollView>
            <Text style={styles.modalTitle}>{s.name}</Text>
            <Text style={styles.rowSub}>{s.district} · {s.road_type}</Text>
            <View style={styles.metrics}>
              <Metric label="Risk" value={`${s.risk_score}`} color={riskColor(s.risk_level)} />
              <Metric label="Lights" value={`${s.street_light_count}`} />
              <Metric label="Adequacy" value={`${Math.round(s.adequacy * 100)}%`} />
              <Metric label="Frames" value={`${detail.analyses.length}`} />
            </View>
            <Text style={styles.kvkk}>
              KVKK: {blurred} faces/plates irreversibly blurred · only urban assets
              detected · no raw imagery stored.
            </Text>
          </ScrollView>
          <Pressable style={styles.closeBtn} onPress={onClose}>
            <Text style={styles.closeBtnText}>Close</Text>
          </Pressable>
        </View>
      </View>
    </Modal>
  );
}

function ReportScreen() {
  const [result, setResult] = useState<AnalyzeResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<string>("");

  async function analyzeHere() {
    setLoading(true);
    setStatus("Getting your location…");
    setResult(null);
    try {
      const perm = await Location.requestForegroundPermissionsAsync();
      if (perm.status !== "granted") {
        setStatus("Location permission denied.");
        return;
      }
      const loc = await Location.getCurrentPositionAsync({});
      setStatus("Analyzing street lighting…");
      const hour = new Date().getHours();
      const res = await api.analyze({
        lat: loc.coords.latitude,
        lon: loc.coords.longitude,
        is_night: hour >= 19 || hour < 7,
        address: "Field report",
      });
      setResult(res);
      setStatus("");
    } catch (e) {
      setStatus(String(e));
    } finally {
      setLoading(false);
    }
  }

  return (
    <ScrollView contentContainerStyle={styles.reportWrap}>
      <Text style={styles.reportLead}>
        Standing under a poorly-lit street? Capture a lighting assessment of your
        exact location. Faces and plates are anonymized automatically.
      </Text>

      <Pressable style={styles.cta} onPress={analyzeHere} disabled={loading}>
        {loading ? (
          <ActivityIndicator color="#0b1120" />
        ) : (
          <Text style={styles.ctaText}>Analyze my location</Text>
        )}
      </Pressable>

      {!!status && <Text style={styles.statusText}>{status}</Text>}

      {result && (
        <View style={styles.resultCard}>
          <View style={styles.resultHeader}>
            <View style={[styles.riskChip, { backgroundColor: riskColor(result.risk_level) }]} />
            <Text style={styles.resultRisk}>
              {result.risk_level.toUpperCase()} · {result.risk_score}
            </Text>
            <Text style={styles.resultSource}>via {result.source}</Text>
          </View>
          <View style={styles.metrics}>
            <Metric label="Lights" value={`${result.street_light_count}`} />
            <Metric label="Poles" value={`${result.pole_count}`} />
            <Metric label="Adequacy" value={`${Math.round(result.adequacy * 100)}%`} />
          </View>
          <Text style={styles.kvkk}>
            Detector: {result.detector_backend} · {result.faces_blurred + result.plates_blurred}{" "}
            faces/plates blurred · KVKK compliant.
          </Text>
        </View>
      )}
    </ScrollView>
  );
}

function Metric({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <View style={styles.metric}>
      <Text style={[styles.metricValue, color ? { color } : null]}>{value}</Text>
      <Text style={styles.metricLabel}>{label}</Text>
    </View>
  );
}

function Centered({ children }: { children: React.ReactNode }) {
  return <View style={styles.centered}>{children}</View>;
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.bg },
  header: { flexDirection: "row", alignItems: "center", gap: 10, padding: 16, paddingBottom: 8 },
  dot: { width: 12, height: 12, borderRadius: 6, backgroundColor: colors.accent },
  title: { color: colors.text, fontSize: 20, fontWeight: "700" },
  subtitle: { color: colors.textDim, fontSize: 12 },
  tabs: { flexDirection: "row", paddingHorizontal: 16, gap: 8, paddingBottom: 8 },
  tab: { flex: 1, paddingVertical: 8, borderRadius: 10, borderWidth: 1, borderColor: colors.border, alignItems: "center" },
  tabActive: { backgroundColor: colors.card, borderColor: colors.accent },
  tabText: { color: colors.textDim, fontSize: 13 },
  tabTextActive: { color: colors.text, fontWeight: "600" },
  map: { height: 260, width: "100%" },
  list: { flex: 1, paddingHorizontal: 16 },
  listHeader: { color: colors.textDim, fontSize: 12, paddingVertical: 8, textTransform: "uppercase" },
  row: { flexDirection: "row", alignItems: "center", gap: 10, paddingVertical: 10, borderBottomWidth: 1, borderBottomColor: colors.border },
  riskChip: { width: 10, height: 10, borderRadius: 5 },
  rowTitle: { color: colors.text, fontSize: 14, fontWeight: "500" },
  rowSub: { color: colors.textDim, fontSize: 12 },
  rowScore: { fontSize: 16, fontWeight: "700" },
  centered: { flex: 1, alignItems: "center", justifyContent: "center", padding: 24 },
  error: { color: "#fca5a5", textAlign: "center" },
  modalWrap: { flex: 1, justifyContent: "flex-end", backgroundColor: "rgba(0,0,0,0.5)" },
  modalCard: { backgroundColor: colors.card, borderTopLeftRadius: 20, borderTopRightRadius: 20, padding: 20, maxHeight: "70%" },
  modalHandle: { alignSelf: "center", width: 40, height: 4, borderRadius: 2, backgroundColor: colors.border, marginBottom: 12 },
  modalTitle: { color: colors.text, fontSize: 18, fontWeight: "700" },
  metrics: { flexDirection: "row", gap: 10, marginVertical: 16 },
  metric: { flex: 1, backgroundColor: colors.bg, borderRadius: 10, padding: 10, borderWidth: 1, borderColor: colors.border },
  metricValue: { color: colors.text, fontSize: 18, fontWeight: "700" },
  metricLabel: { color: colors.textDim, fontSize: 11, marginTop: 2 },
  kvkk: { color: colors.textDim, fontSize: 11, lineHeight: 16, backgroundColor: colors.bg, padding: 10, borderRadius: 8 },
  closeBtn: { marginTop: 12, alignItems: "center", paddingVertical: 12, borderRadius: 10, backgroundColor: colors.border },
  closeBtnText: { color: colors.text, fontWeight: "600" },
  reportWrap: { padding: 20, gap: 16 },
  reportLead: { color: colors.textDim, fontSize: 14, lineHeight: 20 },
  cta: { backgroundColor: colors.accent, paddingVertical: 16, borderRadius: 12, alignItems: "center" },
  ctaText: { color: "#0b1120", fontSize: 16, fontWeight: "700" },
  statusText: { color: colors.textDim, textAlign: "center" },
  resultCard: { backgroundColor: colors.card, borderRadius: 16, padding: 16, borderWidth: 1, borderColor: colors.border },
  resultHeader: { flexDirection: "row", alignItems: "center", gap: 8 },
  resultRisk: { color: colors.text, fontSize: 16, fontWeight: "700" },
  resultSource: { color: colors.textDim, fontSize: 12, marginLeft: "auto" },
});
