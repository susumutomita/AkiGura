#!/usr/bin/env python3
"""
スクレイピングして施設一覧を取得し、マスターデータ用SQLを生成する。

Usage:
    python list_facilities.py [scraper_type]
    python list_facilities.py --all
    python list_facilities.py --sql  # SQL形式で出力
"""

import json
import os
import re
import sys
import uuid
from collections import defaultdict
from datetime import datetime
from typing import Dict, List, Set, Any

# Configure all scrapers with wide time range
ALL_WEEKDAYS = "月曜日,火曜日,水曜日,木曜日,金曜日,土曜日,日曜日,祝日"
for prefix in ("HIRATSUKA", "AYASE", "YOKOHAMA"):
    os.environ.setdefault(f"{prefix}_TIME_FROM", "0000")
    os.environ.setdefault(f"{prefix}_TIME_TO", "2359")
    os.environ.setdefault(f"{prefix}_SELECTED_WEEK_DAYS", ALL_WEEKDAYS)

script_dir = os.path.dirname(os.path.abspath(__file__))
ground_reservation_path = os.path.join(script_dir, '..', '..', 'ground-reservation')
sys.path.insert(0, ground_reservation_path)

try:
    from app.facilities.yokohama.yokohama_facility import YokohamaFacility
    from app.facilities.ayase.ayase_facility import AyaseFacility
    from app.facilities.hiratsuka.hiratsuka_facility import HiratsukaFacility
    from app.facilities.kanagawa_system.kanagawa.kanagawa_facility import KanagawaFacility
    from app.facilities.kanagawa_system.kamakura.kamakura_facility import KamakuraFacility
    from app.facilities.kanagawa_system.fujisawa.fujisawa_facility import FujisawaFacility
    SCRAPERS_AVAILABLE = True
except ImportError as e:
    SCRAPERS_AVAILABLE = False
    IMPORT_ERROR = str(e)

# 自治体IDのマッピング（DBと一致させる）
MUNICIPALITY_IDS = {
    "hiratsuka": "1438eb89-e0d1-4b49-b9a2-135718c207e2",
    "fujisawa": "af168e03-6f16-455b-8319-000f6d2e32bf",
    "yokohama": "e7f6a658-4549-4761-8b77-b316576b22d6",
    "kamakura": "c8e4f9a1-2345-4678-9abc-def012345678",
    "kanagawa": "d9e5f0b2-3456-5789-0bcd-ef0123456789",
    "ayase": "e0f1a2b3-4567-6890-1cde-f01234567890",
}

# 自治体名
MUNICIPALITY_NAMES = {
    "hiratsuka": "平塚市",
    "fujisawa": "藤沢市",
    "yokohama": "横浜市",
    "kamakura": "鎌倉市",
    "kanagawa": "神奈川県",
    "ayase": "綾瀬市",
}


def normalize_facility_name(name: str) -> str:
    """施設名を正規化（面番号を除去して基本名を取得）"""
    # 「Ａ面」「A面」「1面」などを除去
    normalized = re.sub(r'[ＡＢＣＤＥＦＧＨＩＪＫＬＭＡａ-ｚA-Za-z0-9０-９]+面$', '', name)
    normalized = re.sub(r'[\(（][^)）]+[\)）]$', '', normalized)  # 括弧を除去
    normalized = normalized.strip()
    return normalized if normalized else name


def extract_facilities_from_scraper(scraper_type: str) -> Set[str]:
    """スクレイパーを実行して施設名一覧を取得"""
    if not SCRAPERS_AVAILABLE:
        print(f"Error: Scrapers not available: {IMPORT_ERROR}", file=sys.stderr)
        return set()
    
    scrapers = {
        "yokohama": YokohamaFacility,
        "ayase": AyaseFacility,
        "hiratsuka": HiratsukaFacility,
        "kanagawa": KanagawaFacility,
        "kamakura": KamakuraFacility,
        "fujisawa": FujisawaFacility,
    }
    
    if scraper_type not in scrapers:
        print(f"Error: Unknown scraper type: {scraper_type}", file=sys.stderr)
        return set()
    
    try:
        print(f"Scraping {scraper_type}...", file=sys.stderr)
        facility = scrapers[scraper_type]()
        raw_results = facility.search_facility()
        
        # 施設名を抽出して正規化
        facilities = set()
        raw_names = set()
        
        for slot_str in (raw_results or []):
            # 施設名を抽出
            court_name = None
            
            # 日本語日付形式: "令和08年02月28日(土) 06:00 ～ 08:00 施設名"
            if match := re.match(r'.*\d+:\d+\s*[～~-]\s*\d+:\d+\s+(.+)', slot_str.strip()):
                court_name = match.group(1).strip()
            # スラッシュ形式: "01/17(土) 13:00 ～ 15:00 施設名"
            elif match := re.match(r'\d+/\d+\([^)]+\)\s+\d+:\d+\s*[～~-]\s*\d+:\d+\s+(.+)', slot_str.strip()):
                court_name = match.group(1).strip()
            
            if court_name:
                raw_names.add(court_name)
                normalized = normalize_facility_name(court_name)
                facilities.add(normalized)
        
        print(f"  Found {len(raw_names)} raw names, {len(facilities)} unique facilities", file=sys.stderr)
        return facilities
        
    except Exception as e:
        print(f"Error scraping {scraper_type}: {e}", file=sys.stderr)
        return set()


def generate_court_pattern(facility_name: str) -> str:
    """施設名からマッチングパターンを生成"""
    # 基本的には施設名の主要部分をパターンとして使用
    # 「野球場」「公園」などの共通部分を除いた固有名詞部分
    pattern = facility_name
    
    # よくある施設タイプを除去して固有名詞部分を抽出
    for suffix in ['野球場', 'グラウンド', '球場', '公園', 'スタジアム', 'サッカー場', '広場']:
        if pattern.endswith(suffix):
            core = pattern[:-len(suffix)]
            if core:  # 空にならなければ
                pattern = core
                break
    
    return pattern


def generate_sql(facilities_by_scraper: Dict[str, Set[str]]) -> str:
    """施設一覧からマイグレーションSQLを生成"""
    lines = [
        "-- 施設マスターデータ（スクレイピングから生成）",
        f"-- Generated: {datetime.now().isoformat()}",
        "",
    ]
    
    for scraper_type, facilities in sorted(facilities_by_scraper.items()):
        if not facilities:
            continue
            
        municipality_id = MUNICIPALITY_IDS.get(scraper_type)
        municipality_name = MUNICIPALITY_NAMES.get(scraper_type)
        
        if not municipality_id:
            print(f"Warning: No municipality ID for {scraper_type}", file=sys.stderr)
            continue
        
        lines.append(f"-- {municipality_name} ({scraper_type})")
        lines.append(f"INSERT OR IGNORE INTO grounds (id, municipality_id, name, court_pattern, enabled, created_at) VALUES")
        
        values = []
        for facility_name in sorted(facilities):
            ground_id = str(uuid.uuid4())
            court_pattern = generate_court_pattern(facility_name)
            values.append(
                f"('{ground_id}', '{municipality_id}', '{facility_name}', '{court_pattern}', 1, CURRENT_TIMESTAMP)"
            )
        
        lines.append(",\n".join(values) + ";")
        lines.append("")
    
    return "\n".join(lines)


def main():
    import argparse
    parser = argparse.ArgumentParser(description="施設一覧をスクレイピングして取得")
    parser.add_argument("scraper_type", nargs="?", help="スクレイパータイプ (hiratsuka, yokohama, etc.)")
    parser.add_argument("--all", action="store_true", help="全スクレイパーを実行")
    parser.add_argument("--sql", action="store_true", help="SQL形式で出力")
    parser.add_argument("--json", action="store_true", help="JSON形式で出力")
    args = parser.parse_args()
    
    if not SCRAPERS_AVAILABLE:
        print(f"Error: Scrapers not available: {IMPORT_ERROR}", file=sys.stderr)
        sys.exit(1)
    
    scraper_types = ["hiratsuka", "fujisawa", "yokohama", "kamakura", "kanagawa", "ayase"]
    
    if args.all:
        target_scrapers = scraper_types
    elif args.scraper_type:
        target_scrapers = [args.scraper_type]
    else:
        print(f"Usage: {sys.argv[0]} <scraper_type> or --all")
        print(f"Available scrapers: {', '.join(scraper_types)}")
        sys.exit(1)
    
    facilities_by_scraper = {}
    for scraper_type in target_scrapers:
        facilities = extract_facilities_from_scraper(scraper_type)
        facilities_by_scraper[scraper_type] = facilities
    
    if args.sql:
        print(generate_sql(facilities_by_scraper))
    elif args.json:
        result = {
            scraper_type: sorted(list(facilities))
            for scraper_type, facilities in facilities_by_scraper.items()
        }
        print(json.dumps(result, ensure_ascii=False, indent=2))
    else:
        # デフォルト: 人間が読める形式
        for scraper_type, facilities in sorted(facilities_by_scraper.items()):
            print(f"\n=== {MUNICIPALITY_NAMES.get(scraper_type, scraper_type)} ({scraper_type}) ===")
            for facility in sorted(facilities):
                pattern = generate_court_pattern(facility)
                print(f"  - {facility}  [pattern: {pattern}]")


if __name__ == "__main__":
    main()
