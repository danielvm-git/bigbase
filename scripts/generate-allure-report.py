import os
from pathlib import Path
import xml.etree.ElementTree as ET
import yaml

def generate_reports():
    root_dir = Path(".")
    epics_dir = root_dir / "specs" / "epics"
    
    # 1. Carrega o execution-status.yaml para saber o status real de cada story
    exec_status_path = root_dir / "specs" / "execution-status.yaml"
    if not exec_status_path.exists():
        print("specs/execution-status.yaml não encontrado. Rode o sync-status primeiro.")
        return
        
    try:
        exec_data = yaml.safe_load(exec_status_path.read_text()) or {}
    except Exception as e:
        print(f"Erro ao ler specs/execution-status.yaml: {e}")
        return
        
    stories_status = exec_data.get("stories") or {}
    
    # Criar pasta para resultados do Allure
    reports_dir = root_dir / "allure-results"
    reports_dir.mkdir(exist_ok=True)
    
    testsuites = ET.Element("testsuites")
    
    # 2. Varre os arquivos de épicos e mapeia os casos de teste
    for yaml_path in sorted(epics_dir.glob("e*.yaml")):
        try:
            data = yaml.safe_load(yaml_path.read_text()) or {}
        except Exception:
            continue
            
        epic_id = data.get("id", "unknown")
        epic_title = data.get("title", "Untitled Epic")
        
        testsuite = ET.SubElement(testsuites, "testsuite", {
            "name": f"{epic_id} - {epic_title}",
            "tests": "0"
        })
        
        test_count = 0
        for story in data.get("stories") or []:
            story_id = story.get("id", "unknown")
            story_title = story.get("title", "Untitled Story")
            
            # Pega o status da story no execution-status.yaml (default: planned)
            story_status = stories_status.get(story_id, "planned")
            
            for task in story.get("tasks") or []:
                test_count += 1
                task_desc = task.get("desc", "Untitled Task")
                
                testcase = ET.SubElement(testsuite, "testcase", {
                    "name": task_desc,
                    "classname": f"{story_id} - {story_title}"
                })
                
                # Mapeamento do status para o Allure (planned/pending/active aparecem como pendentes)
                if story_status in ["planned", "pending"]:
                    ET.SubElement(testcase, "skipped", {
                        "message": f"Story is planned - Not yet implemented."
                    })
                elif story_status == "active":
                    ET.SubElement(testcase, "skipped", {
                        "message": f"Story is active - In development."
                    })
                # Se for "done", o teste conta como Passed (Verde)
        
        testsuite.set("tests", str(test_count))
        
    # Salvar arquivo de resultados JUnit
    output_file = reports_dir / "junit-results.xml"
    tree = ET.ElementTree(testsuites)
    tree.write(output_file, encoding="utf-8", xml_declaration=True)
    print(f"Relatório de progresso BigPowers JUnit gerado com sucesso em: {output_file}")

if __name__ == "__main__":
    generate_reports()
